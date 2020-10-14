package buffer

import (
	"github.com/wogri/bbox/structs/temperature"
	"reflect"
	"testing"
)

func TestBufferAppend(t *testing.T) {
	var bBuffer *Buffer
	bBuffer = new(Buffer)
	temp := temperature.Temperature{
		Temperature: 31.0,
		BBoxID:      "1234asdf",
		SensorID:    "1234asdf",
	}
	bBuffer.AppendTemperature(temp)
	tempSlice := make([]temperature.Temperature, 1)
	tempSlice[0] = temp
	if !reflect.DeepEqual(tempSlice, bBuffer.GetTemperatures()) {
		t.Errorf("Unexpected result after adding Temperature")
	}
}

type HttpClientMock struct {
	Status string
	Error  error
}

func (h *HttpClientMock) PostData(string, interface{}) (string, error) {
	return h.Status, h.Error
}

func TestBufferSuccessfulFlush(t *testing.T) {
	var bBuffer *Buffer
	bBuffer = new(Buffer)
	temp := temperature.Temperature{
		Temperature: 31.0,
		BBoxID:      "1234asdf",
		SensorID:    "1234asdf",
	}
	bBuffer.AppendTemperature(temp)
	expected := make([]temperature.Temperature, 1)
	expected[0] = temp
	result := bBuffer.GetTemperatures()
	if !reflect.DeepEqual(expected, result) {
		t.Errorf("Unexpected result after adding Temperature; expected: \n%v\nvs\n%v", result, expected)
	}
	c := HttpClientMock{"200", nil}
	err := bBuffer.Flush(&c)
	if err != nil {
		t.Errorf("Unexpected result after flushing to success")
	}
	result = bBuffer.GetTemperatures()
	expected = make([]temperature.Temperature, 0)
	if !reflect.DeepEqual(expected, result) {
		t.Errorf(`Unexpected result after successful Flush():
expected: %v
vs
result: %v`, expected, result)
	}
}

func TestBufferFailedFlush(t *testing.T) {
	var bBuffer *Buffer
	bBuffer = new(Buffer)
	temp := temperature.Temperature{
		Temperature: 31.0,
		BBoxID:      "1234asdf",
		SensorID:    "1234asdf",
	}
	bBuffer.AppendTemperature(temp)
	c := HttpClientMock{"501", &BufferError{}}
	err := bBuffer.Flush(&c)
	if err == nil {
		t.Errorf("Unexpected result after flushing to fail")
	}
	result := bBuffer.GetTemperatures()
	expected := make([]temperature.Temperature, 1)
	expected[0] = temp
	if !reflect.DeepEqual(expected, result) {
		t.Errorf(`Unexpected result after failing Flush():
expected: %v
vs
result: %v`, expected, result)
	}
}
