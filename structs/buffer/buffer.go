package buffer

import (
	"bytes"
	"encoding/json"
	"github.com/wogri/bbox/structs/scale"
	"github.com/wogri/bbox/structs/temperature"
	"log"
	"net/http"
)

type Buffer struct {
	temperatures []temperature.Temperature
	bufferScales []scale.Scale
}
type BufferError struct{}

func (m *BufferError) Error() string {
	return "Could not flush Buffer to API server."
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

func (b *Buffer) String() ([]byte, error) {
	return json.MarshalIndent(b, "", "  ")
}

func postData(apiServer string, token string, data interface{}) (string, error) {
	j, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequest("POST", apiServer, bytes.NewBuffer(j))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Token", token)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	return resp.Status, nil
}

func (b *Buffer) Flush(apiServer string, token string) error {
	for _, t := range b.temperatures {
		status, err := postData(apiServer+"temperature", token, t)
    if err != nil {
      return err
    }
		if status != "200" {
      log.Println("Status: %s", status)
			return &BufferError{}
		}
	}
	// TODO: implement the same shit for scale.
	return nil
}

func (b *Buffer) AppendTemperature(t temperature.Temperature) {
	b.temperatures = append(b.temperatures, t)
}
