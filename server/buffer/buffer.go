package buffer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/queensaver/bbox/server/scheduler"
	"github.com/queensaver/packages/logger"
	"github.com/queensaver/packages/scale"
	"github.com/queensaver/packages/temperature"
)

type Buffer struct {
	temperatures       []temperature.Temperature
	scales             []scale.Scale
	shutdownDesired    bool // If true it will actually physically shutdown the raspberry pi after all data is flushed. It will use the wittypi module to wake up the raspberry pi afterwards.
	temperatureFlushed bool // Set to true if the temperature has been flushed (only useful with shutdownDesired == true)
	scaleFlushed       bool // Set to true if the scale has been flushed (only useful with shutdowDesired  == true)
	schedule           *scheduler.Schedule
	path               string //Path on disk to buffer the data if we can't push it out to the cloud.
}

type BufferError struct {
	message string
}

func (m *BufferError) Error() string {
	return "Could not flush Buffer to API server:" + m.message
}

type HttpClientPoster interface {
	PostData(string, interface{}) error
}

type HttpPostClient struct {
	ApiServer string
	Token     string
}

type DiskBuffer interface {
	Flush(string, string) error
	LoadTemperatures(string) ([]temperature.Temperature, error)
	SaveTemperatures(string, []temperature.Temperature) []temperature.Temperature
	DeleteTemperatures(string, []temperature.Temperature)
	NewFiler(string) *Filer
}

type File struct {
	path string
	lock sync.Mutex
}

type Filer interface {
	Save(interface{}) error
	Load(interface{}) error
	Delete() error
	Path() string
}

var mu sync.Mutex

func (f *File) Save(v interface{}) error {
	f.lock.Lock()
	defer f.lock.Unlock()
	fh, err := os.Create(f.path)
	if err != nil {
		return err
	}
	defer fh.Close()
	r, err := json.Marshal(v)
	if err != nil {
		return err
	}
	_, err = io.Copy(fh, bytes.NewReader(r))
	return err
}

func (f *File) Load(v interface{}) error {
	f.lock.Lock()
	defer f.lock.Unlock()
	fh, err := os.Open(f.path)
	if err != nil {
		return err
	}
	defer fh.Close()
	d, err := ioutil.ReadAll(fh)
	if err != nil {
		return err
	}
	return json.Unmarshal(d, v)
}

func (f *File) Path() string {
	return f.path
}

func (f *File) Delete() error {
	f.lock.Lock()
	defer f.lock.Unlock()
	return os.Remove(f.path)
}

func (b *Buffer) NewFiler(p string) Filer {
	return &File{path: p}
}

func (b *Buffer) DeleteTemperatures(path string, temps []temperature.Temperature) {
	for _, t := range temps {
		f := b.NewFiler(filepath.Join(path, t.UUID+".json"))
		err := f.Delete()
		if err != nil {
			logger.Error("Delete error",
				"filename", f.Path(),
				"error", err,
			)
		}
	}
}

// SaveTemperatures reuturns the temperatures that have NOT been saved to disk to keep them in memory
func (b *Buffer) SaveTemperatures(path string, temps []temperature.Temperature) []temperature.Temperature {
	var unsavedTemperatures []temperature.Temperature
	for _, t := range temps {
		if t.UUID == "" {
			uuid := uuid.New()
			t.UUID = uuid.String()
		}
		f := b.NewFiler(filepath.Join(path, t.UUID+".json"))
		err := f.Save(t)
		if err != nil {
			logger.Error("could not save temperature.", "error", err)
			unsavedTemperatures = append(unsavedTemperatures, t)
		}
	}
	return unsavedTemperatures
}

func (b *Buffer) LoadTemperatures(path string) ([]temperature.Temperature, error) {
	var r []temperature.Temperature
	err := filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if filepath.Ext(path) != ".json" {
			return nil
		}
		if info.IsDir() {
			return nil
		}
		t := temperature.Temperature{}
		f := b.NewFiler(p)
		err = f.Load(t)
		if err != nil {
			r = append(r, t)
			logger.Error("could not load temperature file from disk",
				"path", p,
				"error", err,
			)
			return err
		}
		return nil
	})
	if err != nil {
		logger.Error("could not load temperatures from disk",
			"path", path,
			"error", err,
		)
	}
	return r, nil
}

func (b *Buffer) remountro() {
	// os.exec("sudo mount -o remount,ro /")
}

func (b *Buffer) remountrw() error {
	// os.exec("sudo mount -o remount,rw /")
	return nil
}
func (h HttpPostClient) PostData(request string, data interface{}) error {
	j, err := json.Marshal(data)
	if err != nil {
		return err
	}
	if !strings.HasSuffix(h.ApiServer, "/") {
		h.ApiServer = h.ApiServer + "/"
	}
	url := h.ApiServer + url.PathEscape(request)
	logger.Debug("none", fmt.Sprintf("Post Request for API Server %s", url))
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(j))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Q-Token", h.Token)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return &BufferError{fmt.Sprintf("HTTP return code: %s; URL: %s", resp.Status, url)}
	}
	return nil
}

func (b *Buffer) String() string {
	//r, _ := json.MarshalIndent(b, "", "  ")
	//return string(r[:])
	return fmt.Sprintf("%v\n%v", b.temperatures, b.scales)
}

func (b *Buffer) SetSchedule(schedule *scheduler.Schedule) {
	b.schedule = schedule
}

func (b *Buffer) SetShutdownDesired(s bool) {
	b.shutdownDesired = s
}

// FlushSchedule will wait for the given duration of seconds and then flush the buffer.
// It will be started as a go routine and retry to flush the buffer.
func (b *Buffer) FlushSchedule(apiServerAddr *string, token string, seconds int) {
	poster := HttpPostClient{*apiServerAddr, token}
	for {
		logger.Debug("none", fmt.Sprintf("sleeping for %d seconds", seconds))
		time.Sleep(time.Duration(seconds) * time.Second)
		err := b.Flush("none", poster)
		if err != nil {
			logger.Error("none", err)
		} else {
			logger.Debug("none", "Sending Data to API server was successful.")
		}
	}
}

func (b *Buffer) Flush(ip string, poster HttpClientPoster) error {
	mu.Lock()
	defer mu.Unlock()
	logger.Debug(ip, "Flushing")
	temperaturesOnDisk, err := b.LoadTemperatures(b.path)
	if err != nil {
		logger.Error("Could not load data from disk", "error", err)
	}
	// copy the temperatures from the buffer
	var temperatures = make([]temperature.Temperature, len(b.temperatures)+len(temperaturesOnDisk))
	var postedTemperatures []temperature.Temperature
	for i, t := range append(b.temperatures, temperaturesOnDisk...) {
		temperatures[i] = t
	}
	// empty the slice.
	b.temperatures = make([]temperature.Temperature, 0)
	var last_err error
	for _, t := range temperatures {
		err := poster.PostData("v1/temperature", t)
		if err != nil {
			last_err = err
			b.temperatures = append(b.temperatures, t)
		} else {
			// If there UUID is not empty this means that the temperature was loaded from disk, hence we have to delete it later in a batch when we remount the disk writeable.
			if t.UUID != "" {
				postedTemperatures = append(postedTemperatures, t)
			}
		}
		if b.shutdownDesired {
			b.temperatureFlushed = true
		}
	}
	err = b.remountrw()
	if err == nil {
		b.temperatures = b.SaveTemperatures(filepath.Join(b.path, "temperatures"), b.temperatures)
		b.DeleteTemperatures(b.path, postedTemperatures)
	}
	b.remountro()

	// Repeat the same thing as above with scale.
	// While we could write a function to DRY I think it's OK if I copy this.
	var scales = make([]scale.Scale, len(b.scales))
	for i, s := range b.scales {
		scales[i] = s
	}
	// empty the slice.
	b.scales = make([]scale.Scale, 0)
	for _, s := range scales {
		err := poster.PostData("v1/scale", s)
		if err != nil {
			last_err = err
			b.scales = append(b.scales, s)
		}
		if b.shutdownDesired {
			b.scaleFlushed = true
		}
	}
	if b.shutdownDesired && b.temperatureFlushed && b.scaleFlushed {
		res := b.schedule.Shutdown()
		if !res {
			b.scaleFlushed = false
			b.temperatureFlushed = false
		}
	}

	return last_err
}

func (b *Buffer) AppendScale(s scale.Scale) {
	mu.Lock()
	defer mu.Unlock()
	b.scales = append(b.scales, s)
}

func (b *Buffer) AppendTemperature(t temperature.Temperature) {
	mu.Lock()
	defer mu.Unlock()
	b.temperatures = append(b.temperatures, t)
}

func (b *Buffer) GetTemperatures() []temperature.Temperature {
	mu.Lock()
	defer mu.Unlock()
	return b.temperatures
}
