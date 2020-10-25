package scale

import (
	"encoding/json"
)

type Scale struct {
	Weight float64
	BBoxID string //BBoxID is usually the Mac address of the raspberry pi in the bHive.
  Timestamp int64
}

func (s *Scale) String() ([]byte, error) {
	return json.MarshalIndent(s, "", "  ")
}
