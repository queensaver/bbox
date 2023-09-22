package main

import (
	"flag"
	"fmt"
	"os"
	"gopkg.in/yaml.v3"
	"text/template"
	"os/exec"
)

var wpaSupplicantTemplate = `ctrl_interface=DIR=/var/run/wpa_supplicant GROUP=netdev
update_config=1
country=DE

network={
	ssid="{{.SSID1}}"
	psk="{{.Password1}}"
	key_mgmt=WPA-PSK
}
network={
	ssid="{{.SSID2}}"
	psk="{{.Password2}}"
	key_mgmt=WPA-PSK
}
`

var configFile = flag.String("config", "/boot/wlan.txt", "path to the config file")
var wpaSupplicantFile = flag.String("wpa_supplicant_file", "/etc/wpa_supplicant/wpa_supplicant.conf", "path to the wpa_supplicant.conf file")

type WlanConfig struct {
	SSID1 string `yaml:"ssid1"`
	Password1 string `yaml:"password1"`
	SSID2 string `yaml:"ssid2"`
	Password2 string `yaml:"password2"`
}

func main () {
	flag.Parse()
	// read the yaml file in the boot directory
	config, err := os.ReadFile(*configFile)
	if err != nil {
		panic(err)
	}
	data := WlanConfig{}
	fmt.Println(string(config))
	err = yaml.Unmarshal(config, &data)
	if err != nil {
		panic(err)
	}

	if data.SSID1 == "" {
		panic("ssid1 is empty")
	}
	if data.Password1 == "" {
		panic("password1 is empty")
	}

	// fill out the template
	t, err := template.New("wpa_supplicant").Parse(wpaSupplicantTemplate)
	if err != nil {
		panic(err)
	}

	f, err := os.Create(*wpaSupplicantFile)
	if err != nil {
		cmd := exec.Command("mount", "-o", "remount,rw", "/")
		err = cmd.Run()
		if err != nil {
			panic(err)
		}
		panic(err)
	}
	defer f.Close()
	err = t.Execute(f, data)
	if err != nil {
		panic(err)
	}
}