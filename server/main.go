package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/btelemetry/bbox/server/relay"
	"github.com/btelemetry/bbox/server/scheduler"
	"github.com/btelemetry/packages/buffer"
	"github.com/btelemetry/packages/config"
	"github.com/btelemetry/packages/logger"
	"github.com/btelemetry/packages/scale"
	"github.com/btelemetry/packages/temperature"
)

var apiServerAddr = flag.String("api_server_addr", "https://api.queensaver.wogri.com", "API Server Address")
var httpServerPort = flag.String("http_server_port", "8333", "HTTP server port")
var httpServerHiveFile = flag.String("http_server_bhive_file", "/home/pi/bOS/bhive", "HTTP server directory to serve bHive file")
var flushInterval = flag.Int("flush_interval", 60, "Interval in seconds when the data is flushed to the bCloud API")
var debug = flag.Bool("debug", false, "debug mode")

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
	s.Timestamp = int64(time.Now().Unix())
	logger.Debug(req.RemoteAddr, fmt.Sprintf("successfully received weight from bHive %s", s.BHiveID))
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

	bConfig, err = config.Get(*apiServerAddr + "/v1/config")
	// TODO: this needs to be downloaded before every scheduler run
	if err != nil {
		log.Fatal(err)
	}
	http.HandleFunc("/scale", scaleHandler)
	http.HandleFunc("/temperature", temperatureHandler)
	http.HandleFunc("/config", configHandler)
	http.HandleFunc("/bhive", func(res http.ResponseWriter, req *http.Request) {
		http.ServeFile(res, req, *httpServerHiveFile)
	})

	var relaySwitches []relay.Switcher
	for _, bhive := range bConfig.BHives {
		relaySwitches = append(relaySwitches, &relay.Switch{Gpio: bhive.RelayGPIO})
	}
	relay := relay.RelayModule{}
	err = relay.Initialize(relaySwitches)
	if err != nil {
		logger.Debug("", fmt.Sprintf("bbox relay problems: %s", err))
	}

	scheduler := scheduler.Schedule{Schedule: "*/5 * * * *", RelayModule: relay}
	c := make(chan bool)
	go scheduler.Start(c)

	go bBuffer.FlushSchedule(apiServerAddr, "token", *flushInterval)

	log.Fatal(http.ListenAndServe(":"+*httpServerPort, nil))
}
