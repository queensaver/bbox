package temperature

import (
	"encoding/json"
)

type Temperature struct {
	Temperature float64
	BBoxID string //BBoxID is usually the Mac address of the raspberry pi in the bHive.
  SensorID string
}

func (t *Temperature) String() ([]byte, error) {
	return json.MarshalIndent(t, "", "  ")
}
