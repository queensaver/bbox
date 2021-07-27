package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/queensaver/bbox/server/relay"
	"github.com/queensaver/bbox/server/scheduler"
	"github.com/queensaver/packages/buffer"
	"github.com/queensaver/packages/config"
	"github.com/queensaver/packages/logger"
	"github.com/queensaver/packages/scale"
	"github.com/queensaver/packages/temperature"
)

var apiServerAddr = flag.String("api_server_addr", "https://api.queensaver.com", "API Server Address")
var httpServerPort = flag.String("http_server_port", "8333", "HTTP server port")
var httpServerHiveFile = flag.String("http_server_bhive_file", "/home/pi/bOS/bhive", "HTTP server directory to serve bHive file")
var flushInterval = flag.Int("flush_interval", 60, "Interval in seconds when the data is flushed to the bCloud API")
var tokenFile = flag.String("token_file", fmt.Sprintf("%s/.queensaver_token",
	os.Getenv("HOME")), "HTTP server port")

var bBuffer buffer.Buffer
var bConfig *config.Config

func temperatureHandler(w http.ResponseWriter, req *http.Request) {
	decoder := json.NewDecoder(req.Body)
	var t temperature.Temperature
	err := decoder.Decode(&t)
	if err != nil {
		logger.Error(req.RemoteAddr, err)
		return
	}
	t.Timestamp = int64(time.Now().Unix())
	bBuffer.AppendTemperature(t)
	logger.Debug(req.RemoteAddr, fmt.Sprintf("successfully received temperature from bHive %s", t.BHiveID))
}

func scaleHandler(w http.ResponseWriter, req *http.Request) {
	decoder := json.NewDecoder(req.Body)
	var s scale.Scale
	err := decoder.Decode(&s)
	if err != nil {
		//logger.Info(err)
		return
	}
	s.Epoch = int64(time.Now().Unix())
	logger.Debug(req.RemoteAddr, fmt.Sprintf("successfully received weight from bHive %s", s.BhiveId))
	bBuffer.AppendScale(s)
}

func configHandler(w http.ResponseWriter, req *http.Request) {
	js, err := json.Marshal(bConfig)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

func main() {
	flag.Parse()
	var err error

	token := os.Getenv("TOKEN")
	if token == "" {
		content, err := ioutil.ReadFile(*tokenFile)
		if err != nil {
			log.Fatal("can't bootstrap without authentication token (set TOKEN environment variable):", err)
		}
		token = string(content)
	}
	bConfig, err = config.Get(*apiServerAddr+"/v1/config", token)
	// TODO: this needs to be downloaded before every scheduler run
	if err != nil {
		log.Fatal(err)
	}
	s, _ := bConfig.String()
	logger.Info("", string(s))
	http.HandleFunc("/scale", scaleHandler)
	http.HandleFunc("/temperature", temperatureHandler)
	http.HandleFunc("/config", configHandler)
	http.HandleFunc("/bhive", func(res http.ResponseWriter, req *http.Request) {
		http.ServeFile(res, req, *httpServerHiveFile)
	})

	var schedule scheduler.Schedule
	// check if the bhive is a local instance, if so, skip the relay initialisation.
	if len(bConfig.Bhive) == 1 && bConfig.Bhive[0].Local == true {
		schedule = scheduler.Schedule{Schedule: bConfig.Schedule,
			Local:   true,
			WittyPi: bConfig.Bhive[0].WittyPi,
			Token:   token}
	} else {
		var relaySwitches []relay.Switcher
		for _, bhive := range bConfig.Bhive {
			relaySwitches = append(relaySwitches, &relay.Switch{Gpio: bhive.RelayGpio})
		}
		myRelay := relay.RelayModule{}
		err = myRelay.Initialize(relaySwitches)
		if err != nil {
			logger.Debug("", fmt.Sprintf("bbox relay problems: %s", err))
		}

		schedule = scheduler.Schedule{Schedule: bConfig.Schedule, RelayModule: myRelay, Token: token}
	}
	c := make(chan bool)
	go schedule.Start(c)

	go bBuffer.FlushSchedule(apiServerAddr, token, *flushInterval)

	log.Fatal(http.ListenAndServe(":"+*httpServerPort, nil))
}
