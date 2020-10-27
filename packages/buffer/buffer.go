package buffer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/wogri/bbox/packages/logger"
	"github.com/wogri/bbox/packages/scale"
	"github.com/wogri/bbox/packages/temperature"
	"net/http"
  "sync"
	"net/url"
	"strings"
)

type Buffer struct {
	temperatures []temperature.Temperature
	scales       []scale.Scale
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
	/*
	  TODO: Implement.
	  bufferToDisk(string) error
	  readFromDisk(string) error
	  deleteFromDisk(string) error
	*/
}

var mu sync.Mutex

func (h HttpPostClient) PostData(request string, data interface{}) error {
	j, err := json.Marshal(data)
	if err != nil {
		return err
	}
	if !strings.HasSuffix(h.ApiServer, "/") {
		h.ApiServer = h.ApiServer + "/"
	}
	url := h.ApiServer + url.PathEscape(request)
	logger.Info("none", fmt.Sprintf("Post Request for API Server %s", url))
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(j))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Token", h.Token)
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

func (b *Buffer) Flush(ip string, poster HttpClientPoster) error {
  mu.Lock()
  defer mu.Unlock()
	logger.Info(ip, "Flushing")
	var temperatures = make([]temperature.Temperature, len(b.temperatures))
	for i, t := range b.temperatures {
		temperatures[i] = t
	}
	// empty the slice.
	b.temperatures = make([]temperature.Temperature, 0)
	var last_err error
	for _, t := range temperatures {
		err := poster.PostData("temperature", t)
		if err != nil {
			last_err = err
			b.temperatures = append(b.temperatures, t)
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
		err := poster.PostData("scale", s)
		if err != nil {
			last_err = err
			b.scales = append(b.scales, s)
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
