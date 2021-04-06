package temperature

import (
	"encoding/json"
)

type Temperature struct {
	Temperature float64
	// TODO: change! This is the BHiveID, not the BBoxID!
	BHiveID   string //BBoxID is usually the Mac address of the raspberry pi in the bHive.
	SensorID  string
	Timestamp int64
}

func (t *Temperature) String() ([]byte, error) {
	return json.MarshalIndent(t, "", "  ")
}
