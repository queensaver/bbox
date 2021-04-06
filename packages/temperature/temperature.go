package temperature

import (
	"encoding/json"
)

type Temperature struct {
	Temperature float64
	BHiveID   string //BHiveID is usually the Mac address of the raspberry pi in the bHive.
	SensorID  string
	Timestamp int64
}

func (t *Temperature) String() ([]byte, error) {
	return json.MarshalIndent(t, "", "  ")
}
