package buffer

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/queensaver/packages/temperature"
)

type HttpClientMock struct {
	Status string
	Error  error
}

func (h *HttpClientMock) PostData(string, interface{}) error {
	return h.Error
}

type FakeFileOperator struct {
	values []SensorValuer
}

func (f *FakeFileOperator) LoadValues(string, func() SensorValuer) ([]SensorValuer, error) {
	return f.values, nil
}

func (f *FakeFileOperator) SaveValues(p string, v []SensorValuer) []SensorValuer {
	f.values = v
	return []SensorValuer{}
}

func (f *FakeFileOperator) DeleteValues(string, []SensorValuer) {
}

func (f *FakeFileOperator) NewFiler(p string) Filer {
	return &FakeFile{p}
}

type FakeFile struct {
	path string
}

func (f *FakeFile) Save(v interface{}) error {
	return nil
}

func (f *FakeFile) Load(v interface{}) error {
	// test if file is of a certain type and then populate with random data
	return nil
}

func (f *FakeFile) Delete() error {
	return nil
}

func (f *FakeFile) Path() string {
	return f.path
}

func TestBufferAppend(t *testing.T) {
	bBuffer := Buffer{
		FileOperator: &FakeFileOperator{},
	}
	temp := temperature.Temperature{
		Temperature: 31.0,
		BHiveID:     "1234asdf",
		SensorID:    "1234asdf",
	}
	bBuffer.AppendTemperature(temp)
	tempSlice := make([]temperature.Temperature, 1)
	tempSlice[0] = temp
	te := bBuffer.GetTemperatures()
	if diff := cmp.Diff(tempSlice, te); diff != "" {
		t.Errorf("Unexpected result after adding Temperature: %s", diff)
	}
}

func TestBufferSuccessfulFlush(t *testing.T) {
	bBuffer := Buffer{
		FileOperator: &FakeFileOperator{},
		path:         "temperatures",
	}
	temp := temperature.Temperature{
		Temperature: 31.0,
		BHiveID:     "1234asdf",
		SensorID:    "1234asdf",
	}
	bBuffer.AppendTemperature(temp)
	expected := make([]temperature.Temperature, 1)
	expected[0] = temp
	result := bBuffer.GetTemperatures()
	if !cmp.Equal(expected, result) {
		t.Errorf("Unexpected result after adding Temperature; %v", cmp.Diff(result, expected))
	}
	c := HttpClientMock{"200", nil}
	err := bBuffer.Flush("1.2.3.4", &c)
	if err != nil {
		t.Errorf("Unexpected result after flushing to success")
	}
	result = bBuffer.GetTemperatures()
	if len(result) > 0 {
		t.Errorf(`Unexpected result after successful Flush():%v`, result)
	}
}

func TestBufferFailedFlush(t *testing.T) {
	bBuffer := Buffer{
		FileOperator: &FakeFileOperator{},
		path:         "temperatures",
	}
	temp := temperature.Temperature{
		Temperature: 31.0,
		BHiveID:     "1234asdf",
		SensorID:    "1234asdf",
	}
	bBuffer.AppendTemperature(temp)
	c := HttpClientMock{"501", &BufferError{}}
	err := bBuffer.Flush("1.2.3.4", &c)
	if err == nil {
		t.Errorf("Unexpected result after flushing to fail")
	}
	result, _ := bBuffer.FileOperator.LoadValues("", func() SensorValuer { return &temperature.Temperature{} })
	expected := []temperature.Temperature{temp}
	if !cmp.Equal(expected, result) {
		t.Errorf(`Unexpected result after failing Flush(): %v`, cmp.Diff(result, expected))
	}
}

func TestBufferFailedFlushMultiAppend(t *testing.T) {
	filer := newFiler
	defer func() { newFiler = filer }()
	newFiler = func(p string) Filer { return &FakeFile{p} }

	bBuffer := Buffer{
		FileOperator: &FakeFileOperator{},
		path:         "temperatures",
	}

	temp1 := temperature.Temperature{
		Temperature: 31.0,
		BHiveID:     "1234asdf",
		SensorID:    "1234asdf",
	}
	temp2 := temperature.Temperature{
		Temperature: 32.0,
		BHiveID:     "1234asdf",
		SensorID:    "1234asdf",
	}
	temp3 := temperature.Temperature{
		Temperature: 33.0,
		BHiveID:     "1234asdf",
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
	result, _ := bBuffer.FileOperator.LoadTemperatures("")
	expected := make([]temperature.Temperature, 3)
	expected[2] = temp1
	expected[1] = temp2
	expected[0] = temp3
	if diff := cmp.Diff(expected, result); diff != "" {
		t.Errorf("%v", diff)
	}
}
