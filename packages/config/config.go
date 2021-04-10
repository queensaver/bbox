package config

import (
	"encoding/json"
)

type Config struct {
	BBoxID    string // This is usually the Mac address of the raspberry pi in the BBox
	BHives    []BHive
	AuthToken string
}

type BHive struct {
	BHiveID            string  //BHiveID is usually the Mac address of the raspberry pi in the bHive.
	RelayGPIO          int     // The GPIO the relay is configured for.
	ScaleOffset        float64 // The offset in grams we substract from the measurement to tare it.
	ScaleReferenceUnit float64 // The reference unit we divide the measurement by to get the desired unit.
	Cameras            int     // Number of cameras in the BHive
	Schedule           string  // Cron schedule from "github.com/robfig/cron/v3"
}

func (s *Config) String() ([]byte, error) {
	return json.MarshalIndent(s, "", "  ")
}
