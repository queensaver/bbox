package scale

import (
	"encoding/json"
)

type Scale struct {
	Weight float64
	BBoxID string //BBoxID is usually the Mac address of the raspberry pi in the bHive.
}

func (s *Scale) String() ([]byte, error) {
	return json.MarshalIndent(s, "", "  ")
}
