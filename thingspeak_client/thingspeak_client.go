package thingspeak_client

import (
	"bytes"
	"encoding/json"
	"net/http"
)

type ChannelWriter struct {
	Key         string  `json:"key"`
	Weight      float64 `json:"field1"`
	Temperature float64 `json:"field2"`
}

func NewChannelWriter(key string) *ChannelWriter {
	w := new(ChannelWriter)
	w.Key = key
	return w
}

func (w *ChannelWriter) SetTemperature(temperature float64) {
	w.Temperature = temperature
}

func (w *ChannelWriter) SetWeight(weight float64) {
	w.Weight = weight
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
