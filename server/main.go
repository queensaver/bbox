package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/queensaver/bbox/server/buffer"
	"github.com/queensaver/bbox/server/relay"
	"github.com/queensaver/bbox/server/scheduler"
	"github.com/queensaver/openapi/golang/proto/services"
	"github.com/queensaver/packages/config"
	"github.com/queensaver/packages/logger"
	"github.com/queensaver/packages/scale"
	"github.com/queensaver/packages/sound"
	"github.com/queensaver/packages/temperature"
	"github.com/queensaver/packages/varroa"
)

var apiServerAddr = flag.String("api_server_addr", "https://api.queensaver.com", "API Server Address")
var httpServerPort = flag.String("http_server_port", "8333", "HTTP server port")
var httpServerHiveFile = flag.String("http_server_bhive_file", "/home/pi/bOS/bhive", "HTTP server directory to serve bHive file")
var flushInterval = flag.Int("flush_interval", 60, "Interval in seconds when the data is flushed to the bCloud API")
var cachePath = flag.String("cache_path", "bCache", "Cache directory where data will be stored that can't be sent to the cloud.")
var tokenFile = flag.String("token_file", fmt.Sprintf("%s/.queensaver_token",
	os.Getenv("HOME")), "Path to the file containing the token")

var bBuffer buffer.Buffer
var bConfig *config.Config
var token string

func temperatureHandler(w http.ResponseWriter, req *http.Request) {
	decoder := json.NewDecoder(req.Body)
	var t temperature.Temperature
	err := decoder.Decode(&t)
	if err != nil {
		logger.Error(req.RemoteAddr, err)
		return
	}
	if t.Error != "" {
		logger.Info("Temperature measurement error received from BHive",
			"bhive_id", t.BHiveID,
			"error", t.Error)
		// TODO: Saving the error is not implemetend on the cloud side, hence we just log the error and null it here.
		t.Error = ""
	}
	t.Timestamp = int64(time.Now().Unix())
	bBuffer.AppendTemperature(t)
	logger.Debug("Successfully received temperature from bHive",
		"ip", req.RemoteAddr,
		"bhive_id", t.BHiveID)
}

func varroaHandler(w http.ResponseWriter, req *http.Request) {
	const maxSize = 32 << 20
	err := req.ParseMultipartForm(maxSize) // maxMemory 32MB
	if err != nil {
		logger.Info("Could not parse MultiPartFrom in postVarroaImageHandler", "erro", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	bHiveID := req.PostFormValue("bhiveId")

	if bHiveID == "" {
		logger.Info("Missing form values - bhiveId is empty.")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	f, _, err := req.FormFile("scan")
	if err != nil {
		logger.Info("Invalid file upload", "error", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	image, err := io.ReadAll(f)
	if err != nil {
		logger.Info("Can't read image", "error", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	v := varroa.Varroa{
		// see https://github.com/golang/go/issues/9859 if this syntax below seems confusing. it's confusing me as well.
		VarroaScanImagePostRequest: services.VarroaScanImagePostRequest{
			BhiveId: bHiveID,
			Epoch:   int64(time.Now().Unix()),
			Scan:    image}}

	bBuffer.AppendVarroaImage(v)
	w.WriteHeader(http.StatusOK)
}

func soundHandler(w http.ResponseWriter, req *http.Request) {
	decoder := json.NewDecoder(req.Body)
	var s sound.Sound
	err := decoder.Decode(&s)
	if err != nil {
		logger.Error("Sound decode error", "error", err)
		return
	}
	if s.Error != "" {
		logger.Error("Sound measurement error received",
			"bhive_id", s.BhiveId,
			"error", s.Error)
	}

	s.Epoch = int64(time.Now().Unix())
	logger.Debug("Successfully received sound from bHive",
		"ip", req.RemoteAddr,
		"bhive_id", s.BhiveId)
	bBuffer.AppendSound(s)
}

func scaleHandler(w http.ResponseWriter, req *http.Request) {
	decoder := json.NewDecoder(req.Body)
	var s scale.Scale
	err := decoder.Decode(&s)
	if err != nil {
		logger.Error("Scale decode error", "error", err)
		return
	}
	if s.Error != "" {
		logger.Error("Scale measurement error received",
			"bhive_id", s.BhiveId,
			"error", s.Error)
		// TODO: Saving the erorr is not implemetend on the cloud side, hence we just log the error and null it here.
		s.Error = ""
	}

	s.Epoch = int64(time.Now().Unix())
	logger.Debug("Successfully received temperature from bHive",
		"ip", req.RemoteAddr,
		"bhive_id", s.BhiveId)
	bBuffer.AppendScale(s)
}

// Initiates a flush to cloud. This will hold a lock so that no other values will be accepted from bHIves.
func flushHandler(w http.ResponseWriter, req *http.Request) {
	poster := buffer.HttpPostClient{ApiServer: *apiServerAddr, Token: token}
	go bBuffer.Flush(poster)
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
	logger.Debug("bbox starting up")

	token = os.Getenv("TOKEN")
	if token == "" {
		content, err := ioutil.ReadFile(*tokenFile)
		if err != nil {
			logger.Fatal("can't bootstrap without authentication token (set TOKEN environment variable):", err)
		}
		token = string(content)
	}
	bConfig, err = config.Get(*apiServerAddr+"/v1/config", token)
	// TODO: this needs to be downloaded before every scheduler run
	if err != nil {
		logger.Fatal("Could not get config", "error", err)
	}
	s, _ := bConfig.String()
	logger.Debug("bConfig content", "bconfig", s)

	// Initiatlize the bBuffer
	bBuffer.SetPath(*cachePath)
	bBuffer.SetFileOperator(&buffer.FileSurgeon{})

	var schedule scheduler.Schedule
	// check if the bhive is a local instance, if so, skip the relay initialisation.
	if (len(bConfig.Bhive) == 1) && bConfig.Bhive[0].Local {
		witty := bConfig.Bhive[0].WittyPi
		schedule = scheduler.Schedule{Schedule: bConfig.Schedule,
			Local:   true,
			WittyPi: witty,
			Token:   token}

		bBuffer.SetSchedule(&schedule)
		if witty {
			bBuffer.SetShutdownDesired(true)
		}
		go bBuffer.FlushSchedule(apiServerAddr, token, *flushInterval)
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
		go bBuffer.FlushSchedule(apiServerAddr, token, *flushInterval)
	}
	c := make(chan bool)
	go schedule.Start(c)

	http.HandleFunc("/scale", scaleHandler)
	http.HandleFunc("/temperature", temperatureHandler)
	http.HandleFunc("/sound", soundHandler)
	http.HandleFunc("/varroa", varroaHandler)
	http.HandleFunc("/config", configHandler)
	http.HandleFunc("/flush", flushHandler)
	http.HandleFunc("/bhive", func(res http.ResponseWriter, req *http.Request) {
		http.ServeFile(res, req, *httpServerHiveFile)
	})

	log.Fatal(http.ListenAndServe(":"+*httpServerPort, nil))
}
