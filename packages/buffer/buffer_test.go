package buffer

import (
	"github.com/wogri/bbox/packages/temperature"
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

func (h *HttpClientMock) PostData(string, interface{}) error {
	return h.Error
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
	err := bBuffer.Flush("1.2.3.4", &c)
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
	err := bBuffer.Flush("1.2.3.4", &c)
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

func TestBufferFailedFlushMultiAppend(t *testing.T) {
	var bBuffer *Buffer
	bBuffer = new(Buffer)
	temp1:= temperature.Temperature{
		Temperature: 31.0,
		BBoxID:      "1234asdf",
		SensorID:    "1234asdf",
	}
	temp2:= temperature.Temperature{
		Temperature: 32.0,
		BBoxID:      "1234asdf",
		SensorID:    "1234asdf",
	}
	temp3:= temperature.Temperature{
		Temperature: 33.0,
		BBoxID:      "1234asdf",
		SensorID:    "1234asdf",
	}
	bBuffer.AppendTemperature(temp1)
	c := HttpClientMock{"501", &BufferError{}}
	err := bBuffer.Flush("1.2.3.4", &c)
	if err == nil {
		t.Errorf("Unexpected result after flushing to fail")
	}
	bBuffer.AppendTemperature(temp2)
	err = bBuffer.Flush("1.2.3.4", &c)
	if err == nil {
		t.Errorf("Unexpected result after flushing to fail")
	}
	bBuffer.AppendTemperature(temp3)
	err = bBuffer.Flush("1.2.3.4", &c)
	if err == nil {
		t.Errorf("Unexpected result after flushing to fail")
  }
	result := bBuffer.GetTemperatures()
	expected := make([]temperature.Temperature, 3)
	expected[0] = temp1
	expected[1] = temp2
	expected[2] = temp3
	if !reflect.DeepEqual(expected, result) {
		t.Errorf(`Unexpected result after failing Flush() with multiple Temperatures:
expected: %v
vs
result: %v`, expected, result)
	}
}
