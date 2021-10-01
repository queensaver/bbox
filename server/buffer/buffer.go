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
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/queensaver/bbox/server/scheduler"
	"github.com/queensaver/packages/logger"
	"github.com/queensaver/packages/scale"
	"github.com/queensaver/packages/temperature"
)

type Buffer struct {
	unsentTemperatures []SensorValuer
	unsentScaleValues  []SensorValuer
	shutdownDesired    bool // If true it will actually physically shutdown the raspberry pi after all data is flushed. It will use the wittypi module to wake up the raspberry pi afterwards.
	schedule           *scheduler.Schedule
	path               string //Path on disk to buffer the data if we can't push it out to the cloud.
	FileOperator
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

type FileSurgeon struct {
	writeableFS bool
}

type FileOperator interface {
	LoadValues(string, func() SensorValuer) []SensorValuer
	SaveValues(string, []SensorValuer) []SensorValuer
	DeleteValues(string, []SensorValuer)
	RemountRO() error
	RemountRW() error
	NewFiler(string) Filer
}

type SensorValuer interface {
	SetUUID()
	GetUUID() string
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
	logger.Debug("Deleting file", "filename", f.Path())
	return os.Remove(f.path)
}

func (f *FileSurgeon) NewFiler(p string) Filer {
	// return newFiler(p)
	return &File{path: p}

}

func (f *FileSurgeon) DeleteValues(path string, values []SensorValuer) {
	logger.Debug("Deleting Values from Disk")

	if !f.writeableFS {
		err := f.RemountRW()
		if err != nil {
			return
		}
	}

	for _, v := range values {
		f := f.NewFiler(filepath.Join(path, v.GetUUID()+".json"))
		err := f.Delete()
		if err != nil {
			logger.Error("Delete error",
				"filename", f.Path(),
				"error", err,
				"object", v)
		}
	}
}

// SaveValues returns the values could NOT been saved to disk so we can keep them in memory - just in case the disk is full or whatever we're trying to make sure to not lose any data.
func (f *FileSurgeon) SaveValues(path string, values []SensorValuer) []SensorValuer {
	var unsavedValues []SensorValuer
	if !f.writeableFS {
		err := f.RemountRW()
		if err != nil {
			return values
		}
	}
	err := os.MkdirAll(path, 0755)
	if err != nil {
		logger.Error("Could not create directory",
			"path", path,
			"error", err)
		return values
	}
	for _, v := range values {
		if u := v.GetUUID(); u == "" {
			v.SetUUID()
		}
		f := f.NewFiler(filepath.Join(path, v.GetUUID()+".json"))
		err := f.Save(v)
		if err != nil {
			logger.Error("could not save object.",
				"error", err,
				"object", v)
			unsavedValues = append(unsavedValues, v)
		}
	}
	return unsavedValues
}

func (f *FileSurgeon) LoadValues(path string, newObject func() SensorValuer) []SensorValuer {
	logger.Debug("Loading values from disk", "path", path)
	r := []SensorValuer{}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		logger.Debug("Path does not exist", "path", path)
		return r
	}
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
		o := newObject()
		f := f.NewFiler(p)
		err = f.Load(&o)
		if err != nil {
			logger.Error("could not load object file from disk",
				"path", p,
				"error", err,
				"object", o,
			)
			return err
		} else {
			r = append(r, o)
		}
		return nil
	})
	if err != nil {
		logger.Error("could not load objects from disk",
			"path", path,
			"error", err,
		)
	}
	return r
}

func (b *Buffer) SetPath(p string) {
	b.path = p
}

func (b *Buffer) SetFileOperator(o FileOperator) {
	b.FileOperator = o
}

func (f *FileSurgeon) RemountRO() error {
	if !f.writeableFS {
		return nil
	}
	logger.Info("Remounting filesystem read-only")
	cmd := exec.Command("/usr/bin/mount", "-o", "remount,ro", "/")
	err := cmd.Run()
	if err != nil {
		logger.Error("Remount to read-only Erorr", "error", err)
		return err
	}
	f.writeableFS = false
	return nil
}

func (f *FileSurgeon) RemountRW() error {
	logger.Info("Remounting filesystem read-write")
	cmd := exec.Command("/usr/bin/mount", "-o", "remount,rw", "/")
	err := cmd.Run()
	if err != nil {
		logger.Error("Remount to read/write Erorr", "error", err)
		return err
	}
	f.writeableFS = true
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
	logger.Debug("Post Request for API Server", "url", url)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(j))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Q-Token", h.Token)
	client := &http.Client{Timeout: 13 * time.Minute} // TODO: This needs tuning, some documents like images might take longer to upload.
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
	return fmt.Sprintf("%v\n%v", b.unsentTemperatures, b.unsentScaleValues)
}

func (b *Buffer) SetSchedule(schedule *scheduler.Schedule) {
	b.schedule = schedule
}

func (b *Buffer) SetShutdownDesired(s bool) {
	b.shutdownDesired = s
}

func (b *Buffer) ShutdownDesired() bool {
	return b.shutdownDesired
}

// FlushSchedule will wait for the given duration of seconds and then flush the buffer.
// It will be started as a go routine and retry to flush the buffer.
func (b *Buffer) FlushSchedule(apiServerAddr *string, token string, seconds int) {
	poster := HttpPostClient{*apiServerAddr, token}
	for {
		logger.Debug("sleeping", "seconds", seconds)
		time.Sleep(time.Duration(seconds) * time.Second)
		b.Flush(poster)
	}
}

// SendValues will send all []SensorValuer to the given apiServerAddr.
// SendValues will also load cached values from disk and try to send them all to the server in one batch.
// If values from disk are sent out successfully they will be deleted from disk.
// If it can't connect to the apiServer, it will return the values that could neither be sent to the apiServer nor cached to disk.
func (b *Buffer) SendValues(
	path string,
	apiPath string,
	newValues []SensorValuer,
	poster HttpClientPoster,
	newValue func() SensorValuer) ([]SensorValuer, error) {

	valuesOnDisk := b.FileOperator.LoadValues(path, newValue)
	// copy the temperatures from the buffer
	var values = make([]SensorValuer, len(newValues)+len(valuesOnDisk))

	for i, t := range append(newValues, valuesOnDisk...) {
		values[i] = t
	}

	var postedValues []SensorValuer
	var unsentValues []SensorValuer
	for _, v := range values {
		logger.Debug("Sending value",
			"value", v,
			"api_path", apiPath)
		err := poster.PostData(apiPath, v)
		if err != nil {
			logger.Info("error posting data to the cloud", "err", err)
			unsentValues = append(unsentValues, v)
		} else {
			// If the UUID is not empty this means that the value was loaded from disk, hence we have to delete it later in a batch when we remount the disk writeable.
			if v.GetUUID() != "" {
				postedValues = append(postedValues, v)
				logger.Info("Planning to delete UUID.", "path", path, "uuid", v.GetUUID())
			}
		}
	}
	if len(postedValues) > 0 {
		b.FileOperator.DeleteValues(path, postedValues)
	}
	if len(unsentValues) > 0 {
		unsavedValues := b.FileOperator.SaveValues(path, unsentValues)
		return unsavedValues, nil
	}
	return nil, nil
}

func (b *Buffer) Flush(poster HttpClientPoster) {
	logger.Info("Flushing data to cloud")
	defer logger.Info("Flushing data to cloud done")
	mu.Lock()
	defer b.FileOperator.RemountRO()
	defer mu.Unlock()

	var err error
	temperaturePath := filepath.Join(b.path, "temperatures")
	newTemperature := func() SensorValuer { return &scale.Scale{} }
	b.unsentTemperatures, err = b.SendValues(
		temperaturePath,
		"v1/temperature",
		b.unsentTemperatures,
		poster,
		newTemperature)
	if err != nil {
		logger.Error("Could not send temperature values", "error", err)
	}
	scalePath := filepath.Join(b.path, "scale-values")
	newScale := func() SensorValuer { return &scale.Scale{} }
	b.unsentScaleValues, err = b.SendValues(
		scalePath,
		"v1/scale",
		b.unsentScaleValues,
		poster,
		newScale)
	if err != nil {
		logger.Error("Could not send scale values", "error", err)
	}

	if b.ShutdownDesired() && (len(b.unsentScaleValues) == 0) && (len(b.unsentTemperatures) == 0) {
		logger.Info("Shutdown is desired, all data was flushed, attempting to shut down now")
		b.schedule.Shutdown()
	}
}

/*func (b *Buffer) Flush(ip string, poster HttpClientPoster) error {
	mu.Lock()
	defer mu.Unlock()
	logger.Debug(ip, "Flushing")
	temperaturePath := filepath.Join(b.path, "temperatures")
	temperaturesOnDisk, err := b.FileOperator.LoadValues(temperaturePath, func() SensorValuer { return &temperature.Temperature{} })
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
			if t.GetUUID() != "" {
				postedTemperatures = append(postedTemperatures, t)
			}
		}
		if b.shutdownDesired {
			b.temperatureFlushed = true
		}
	}
	err = b.remountrw()
	if err == nil && (len(b.temperatures) > 0 || (len(postedTemperatures)) > 0) {
		b.temperatures = b.FileOperator.SaveValues(temperaturePath, b.temperatures)
		b.FileOperator.DeleteValues(temperaturePath, postedTemperatures)
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
*/

func (b *Buffer) AppendScale(s scale.Scale) {
	mu.Lock()
	defer mu.Unlock()
	b.unsentScaleValues = append(b.unsentScaleValues, &s)
}

func (b *Buffer) AppendTemperature(t temperature.Temperature) {
	mu.Lock()
	defer mu.Unlock()
	b.unsentTemperatures = append(b.unsentTemperatures, &t)
}

func (b *Buffer) GetUnsentTemperatures() []SensorValuer {
	mu.Lock()
	defer mu.Unlock()
	return b.unsentTemperatures
}
