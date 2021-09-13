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
	"strings"
	"sync"
	"time"

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
	lock               sync.Mutex
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
	Save(string, interface{}) error
	Load(string, interface{}) error
	Delete(string) error
}

var mu sync.Mutex

func (b *Buffer) Save(path string, v interface{}) error {
	b.lock.Lock()
	defer b.lock.Unlock()
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	r, err := json.Marshal(v)
	if err != nil {
		return err
	}
	_, err = io.Copy(f, bytes.NewReader(r))
	return err
}

func (b *Buffer) Load(path string, v interface{}) error {
	b.lock.Lock()
	defer b.lock.Unlock()
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	d, err := ioutil.ReadAll(f)
	if err != nil {
		return err
	}
	return json.Unmarshal(d, v)
}

func (b *Buffer) Delete(path string) error {
	b.lock.Lock()
	defer b.lock.Unlock()
	return os.Remove(path)
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
	var temperatures = make([]temperature.Temperature, len(b.temperatures))
	for i, t := range b.temperatures {
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
		}
		if b.shutdownDesired {
			b.temperatureFlushed = true
		}
	}

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
