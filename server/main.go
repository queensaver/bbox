package main

import (
	"bytes"
	"crypto/tls"
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"

	"github.com/queensaver/bbox/server/buffer"
	// "github.com/queensaver/bbox/server/relay"
	"github.com/queensaver/bbox/server/scheduler"
	"github.com/queensaver/openapi/golang/proto/models"
	"github.com/queensaver/openapi/golang/proto/services"
	"github.com/queensaver/packages/config"
	"github.com/queensaver/packages/logger"
	"github.com/queensaver/packages/scale"
	"github.com/queensaver/packages/sound"
	"github.com/queensaver/packages/telemetry"
	"github.com/queensaver/packages/temperature"
	"github.com/queensaver/packages/varroa"
)

var apiServerAddr = flag.String("api_server_addr", "https://api.queensaver.com", "API Server Address")
var httpServerPort = flag.String("http_server_port", "8333", "HTTP server port")
var httpServerHiveFile = flag.String("http_server_bhive_file", "/home/pi/bOS/bhive", "HTTP server directory to serve bHive file")
var flushInterval = flag.Int("flush_interval", 60, "Interval in seconds when the data is flushed to the bCloud API")
var cachePath = flag.String("cache_path", "bCache", "Cache directory where data will be stored that can't be sent to the cloud.")
var scanCmd = flag.String("scan_command", "/home/pi/capture.sh", "Command to execute for a varroa scan.")
var staticFilePath = flag.String("static_file_path", "/home/pi/bOS/webapp", "The path for serving static files")
var registrationIdFile = flag.String("registration_id_file", fmt.Sprintf("%s/.queensaver_registration_id",
	os.Getenv("HOME")), "Path to the file containing the token")

var bBuffer buffer.Buffer
var bConfig *config.Config
var token string

//go:embed webapp.tar.bz2
var webApp []byte

func getMacAddress() (string, error) {
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
			"bhive_id", t.BhiveId,
			"error", t.Error)
		// TODO: Saving the error is not implemetend on the cloud side, hence we just log the error and null it here.
		t.Error = ""
	}
	t.Epoch = int64(time.Now().Unix())
	telemetry := telemetry.Telemetry{}
	telemetry.Values = []*models.Telemetry{
		{
			T: t.Temperature.Temperature,
			M: t.BhiveId,
			E: t.Epoch,
		},
	}
	bBuffer.AppendTelemetry(telemetry)
	logger.Debug("Successfully received temperature from bHive",
		"ip", req.RemoteAddr,
		"bhive_id", t.BhiveId)
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

	logger.Debug("Successfully received varroa image from bHive")
	bBuffer.AppendVarroaImage(v)
	w.WriteHeader(http.StatusOK)
	logger.Debug("Successfully received varroa image from bHive",
		"ip", req.RemoteAddr,
		"bhive_id", bHiveID)
	logger.Debug("Flushing varroa image to cloud", "api_server_address", *apiServerAddr)
	poster := buffer.HttpPostClient{ApiServer: *apiServerAddr, Token: token}
	bBuffer.Flush(poster)

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
	telemetry := telemetry.Telemetry{}
	telemetry.Values = []*models.Telemetry{
		{
			W: float32(s.Weight),
			M: s.BhiveId,
			E: s.Epoch,
		},
	}
	bBuffer.AppendTelemetry(telemetry)
	logger.Debug("Successfully received scale values from bHive",
		"ip", req.RemoteAddr,
		"bhive_id", s.BhiveId,
		"weight", s.Weight)

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

func rootHandler(w http.ResponseWriter, req *http.Request) {
	http.ServeFile(w, req, "/home/pi/index.html")
}

// Serves the static files if it can be found.
// If this is an artificial angular path, just return index.html.
// This removes the requirement to run nginx in front of the service
func indexHandler(w http.ResponseWriter, req *http.Request) {
	logger.Debug("Serving URL", "url", req.URL.RequestURI())
	// check if file exists
	file := path.Join(*staticFilePath, req.URL.Path)
	_, err := os.Stat(file)
	if os.IsNotExist(err) {
		logger.Debug("File not found", "file", file)
		http.ServeFile(w, req, path.Join(*staticFilePath, "index.html"))
		return
	}
	logger.Debug("File found", "file", file)
	staticHandler := http.FileServer(http.Dir(*staticFilePath))
	staticHandler.ServeHTTP(w, req)
}

func scanHandler(w http.ResponseWriter, req *http.Request) {
	cmd := exec.Command(*scanCmd)
	err := cmd.Run()

	if err != nil {
		logger.Error("Scan command failed", "error", err)
		http.Error(w, "Internal Error", 500)
		return
	}
	// return 200
	w.WriteHeader(http.StatusOK)
}

func scanResultHandler(w http.ResponseWriter, req *http.Request) {
	var path = "/home/pi/bOS/scan.jpg"
	img, err := os.Open(path)
	if err != nil {
		logger.Error("could not find image", "error", err)
		http.Error(w, "Internal Error", 500)
		return
	}
	defer img.Close()
	w.Header().Set("Content-Type", "image/jpeg")
	io.Copy(w, img)
}

func main() {
	flag.Parse()
	logger.Debug("bbox starting up")

	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	token = os.Getenv("TOKEN")
	if token == "" {
		logger.Debug("reading registration id from file", "file", *registrationIdFile)
		content, err := os.ReadFile(*registrationIdFile)
		if err != nil {
			logger.Fatal("can't bootstrap without authentication token (set TOKEN environment variable):", err)
		}
		// remove trailing newline
		token = strings.TrimSuffix(string(content), "\n")
	}
	// register the bbox with the cloud
	url := *apiServerAddr + "/v1/configs/bbox/register"

	mac, err := getMacAddress()
	if err != nil {
		logger.Fatal("can't get mac address:", "error", err)
	}
	body := models.Bbox{
		RegistrationId:   token,
		HardwareType:     "smart-socket",
		HardwareRevision: 1,
		BboxId:           mac,
	}
	// convert the body to json
	b, err := json.Marshal(body)
	if err != nil {
		logger.Fatal("can't marshal body:", "error", err)
	}
	// create new reader from json

	bodyReader := bytes.NewReader(b)

	r, err := http.NewRequest("POST", url, bodyReader)
	if err != nil {
		logger.Fatal("Can't post bbox registration parameters", "error", err)
	}

	r.Header.Add("Content-Type", "application/json")
	r.Header.Add("registrationId", token)
	// ignore the TLS certificate

	client := &http.Client{}
	// set client timeout
	client.Timeout = time.Second * 120
	res, err := client.Do(r)
	if err != nil {
		panic(err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusCreated {
		logger.Fatal("can't register bbox:", "status code", res.StatusCode)
	}

	conf := &models.BboxConfigResponse{}
	err = json.NewDecoder(res.Body).Decode(conf)
	if err != nil {
		logger.Fatal("can't decode bbox config:", "error", err)
	}
	logger.Debug("Successfully registered bbox with cloud", "config", conf.String())

	// This is a hack until we have reimplemented the client side / bhive side of the config
	bConfig = &config.Config{}
	if conf.ScaleMeasureInterval < 60 {
		conf.ScaleMeasureInterval = 60 * 60 * 4
	}

	if conf.ScaleMeasureInterval < 3600 {
		bConfig.Schedule = fmt.Sprintf("*/%d * * * *", conf.ScaleMeasureInterval/60)
	} else if conf.ScaleMeasureInterval < 86400 {
		bConfig.Schedule = fmt.Sprintf("0 */%d * * *", conf.ScaleMeasureInterval/3600)
	} else {
		bConfig.Schedule = fmt.Sprintf("0 0 */%d * *", conf.ScaleMeasureInterval/86400)
	}

	logger.Debug("bConfig schedule", "schedule", bConfig.Schedule)
	logger.Debug("tar file", "size", len(webApp))
	if len(webApp) > 0 {
		// write webapp to disk
		err = os.WriteFile("/home/pi/bOS/webapp.tar.bz2", webApp, 0644)
		if err != nil {
			logger.Fatal("can't write webapp to disk", "error", err)
		}
		cmd := exec.Command("tar", "-xjf", "/home/pi/bOS/webapp.tar.bz2", "-C", "/home/pi/bOS", "--strip-components=1")
		err = cmd.Run()
		if err != nil {
			logger.Fatal("can't unpack webapp", "error", err)
		}
	}

	/* old code: get config from cloud
	bConfig, err = config.Get(*apiServerAddr+"/v1/config", token)
	// TODO: this needs to be downloaded before every scheduler run
	if err != nil {
		logger.Fatal("Could not get config", "error", err)
	}
	s, _ := bConfig.String()
	logger.Debug("bConfig content", "bconfig", s)
	*/

	// Initiatlize the bBuffer
	bBuffer.SetPath(*cachePath)
	bBuffer.SetFileOperator(&buffer.FileSurgeon{})

	// check if the bhive is a local instance, if so, skip the relay initialisation.
	//if (len(bConfig.Bhive) == 1) && bConfig.Bhive[0].Local {
	//witty := bConfig.Bhive[0].WittyPi
	schedule := scheduler.Schedule{Schedule: bConfig.Schedule,
		Local:   true,
		WittyPi: false,
		Token:   token}

	bBuffer.SetSchedule(&schedule)
	/*	if witty {
			bBuffer.SetShutdownDesired(true)
		}
	*/
	go bBuffer.FlushSchedule(apiServerAddr, token, int(conf.BatchInterval))
	/*} else {
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
	}*/
	c := make(chan bool)
	go schedule.Start(c)

	http.HandleFunc("/scale", scaleHandler)
	http.HandleFunc("/temperature", temperatureHandler)
	http.HandleFunc("/sound", soundHandler)
	http.HandleFunc("/varroa", varroaHandler)
	http.HandleFunc("/config", configHandler)
	http.HandleFunc("/flush", flushHandler)
	http.HandleFunc("/scan", scanHandler)
	http.HandleFunc("/scanresult", scanResultHandler)
	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/bhive", func(res http.ResponseWriter, req *http.Request) {
		http.ServeFile(res, req, *httpServerHiveFile)
	})

	log.Fatal(http.ListenAndServe(":"+*httpServerPort, nil))
}
