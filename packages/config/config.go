package config

import (
	"encoding/json"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"time"
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

func getMacAddr() (string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}
	a := interfaces[1].HardwareAddr.String()
	if a != "" {
		r := strings.Replace(a, ":", "", -1)
		return r, nil
	}
	return "", nil
}

func Get(addr string) (*Config, error) {
	httpClient := http.Client{
		Timeout: time.Second * 10,
	}

	req, err := http.NewRequest(http.MethodGet, addr, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Auth-Token", "1234")
	mac, err := getMacAddr()
	if err != nil {
		return nil, err
	}
	req.Header.Set("BBoxID", mac)

	res, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	if res.Body != nil {
		defer res.Body.Close()
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	config := Config{}
	err = json.Unmarshal(body, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil

}
