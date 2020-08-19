package thingspeak_client

import (
	"encoding/json"
	"net/http"
	"bytes"
)

type ChannelWriter struct {
  Key    string `json:"key"`
	Field1 float64 `json:"field1"`
}

func NewChannelWriter(key string) *ChannelWriter {
	w := new(ChannelWriter)
	w.Key = key
	return w
}

func (w *ChannelWriter) AddField(n int, value float64) {
	w.Field1 = value
}

func (w *ChannelWriter) Update() (resp *http.Response, err error) {
	requestBody, err := json.Marshal(w)
	if err != nil {
		return nil, err
	}
	client := &http.Client{}
	r, err := http.NewRequest("POST", "https://api.thingspeak.com/update.json", bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, err
	}

	r.Header.Set("Content-type", "application/json")
	resp, err = client.Do(r)
	return resp, err
}
