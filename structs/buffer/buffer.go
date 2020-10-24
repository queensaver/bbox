package buffer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/wogri/bbox/structs/scale"
	"github.com/wogri/bbox/structs/temperature"
	"net/http"
	"path"
)

type Buffer struct {
	temperatures []temperature.Temperature
	scales       []scale.Scale
}

type BufferError struct{
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

func (h HttpPostClient) PostData(request string, data interface{}) error {
	j, err := json.Marshal(data)
	if err != nil {
		return err
	}
	url := path.Join(h.ApiServer, request)
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
  if resp.Status != "200" {
    return &BufferError{fmt.Sprintf("HTTP return code: %s; URL: %s", resp.Status, url)}
  }
	return nil
}

func (b *Buffer) String() string {
	//r, _ := json.MarshalIndent(b, "", "  ")
	//return string(r[:])
	return fmt.Sprintf("%v\n%v", b.temperatures, b.scales)
}

func (b *Buffer) Flush(poster HttpClientPoster) error {
	var temperatures = make([]temperature.Temperature, len(b.temperatures))
	for i, t := range b.temperatures {
		temperatures[i] = t
	}
	// empty the slice.
	b.temperatures = make([]temperature.Temperature, 0)
  var err error
	for _, t := range temperatures {
		err = poster.PostData("temperature", t)
		if err != nil {
			b.temperatures = append(b.temperatures, t)
		}
	}
	return err
	// TODO: implement the same shit for scale.
}

func (b *Buffer) AppendTemperature(t temperature.Temperature) {
	b.temperatures = append(b.temperatures, t)
}

func (b *Buffer) GetTemperatures() []temperature.Temperature {
	return b.temperatures
}
