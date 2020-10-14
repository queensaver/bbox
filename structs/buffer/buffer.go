package buffer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/wogri/bbox/structs/scale"
	"github.com/wogri/bbox/structs/temperature"
	"log"
	"net/http"
	"path"
)

type Buffer struct {
	temperatures []temperature.Temperature
	scales       []scale.Scale
}

type BufferError struct{}

func (m *BufferError) Error() string {
	return "Could not flush Buffer to API server."
}

type HttpClientPoster interface {
	PostData(string, interface{}) (string, error)
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

func (h HttpPostClient) PostData(request string, data interface{}) (string, error) {
	j, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	url := path.Join(h.ApiServer, request)
	log.Println("Posting to ", url)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(j))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Token", h.Token)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	return resp.Status, nil
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
	for _, t := range temperatures {
		status, err := poster.PostData("temperature", t)
		if err != nil {
			log.Println("Error ", err)
			b.temperatures = append(b.temperatures, t)
			return err
		}
		if status != "200" {
			b.temperatures = append(b.temperatures, t)
			log.Println("Status: ", status)
			return &BufferError{}
		}
	}
	// TODO: implement the same shit for scale.
	return nil
}

func (b *Buffer) AppendTemperature(t temperature.Temperature) {
	b.temperatures = append(b.temperatures, t)
}

func (b *Buffer) GetTemperatures() []temperature.Temperature {
	return b.temperatures
}
