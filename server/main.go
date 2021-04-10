package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/wogri/bbox/packages/buffer"
	"github.com/wogri/bbox/packages/config"
	"github.com/wogri/bbox/packages/logger"
	"github.com/wogri/bbox/packages/scale"
	"github.com/wogri/bbox/packages/temperature"
	"github.com/wogri/bbox/server/relay"
	"github.com/wogri/bbox/server/scheduler"
)

var apiServerAddr = flag.String("api_server_addr", "https://bcloud-api.azure.wogri.com", "API Server Address")
var httpServerPort = flag.String("http_server_port", "8333", "HTTP server port")
var httpServerHiveFile = flag.String("http_server_bhive_file", "/home/pi/bOS/bhive", "HTTP server directory to serve bHive file")
var flushInterval = flag.Int("flush_interval", 60, "Interval in seconds when the data is flushed to the bCloud API")
var debug = flag.Bool("debug", false, "debug mode")
var prometheusActive = flag.Bool("prometheus", false, "Activate Prometheus exporter")

var (
	promTemperature = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "bhive_temperature",
		Help: "Temperature of the bHive",
	},
		[]string{"BHiveID", "SensorID"},
	)
	promWeight = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "bhive_weight",
		Help: "Weight of the bHive",
	},
		[]string{"BHiveID"},
	)
)

var bBuffer buffer.Buffer

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
	if *prometheusActive {
		promTemperature.WithLabelValues(t.BHiveID, t.SensorID).Set(t.Temperature)
	}
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
	if *prometheusActive {
		promWeight.WithLabelValues(s.BHiveID).Set(s.Weight)
	}
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

func getConfig() (*config.Config, error) {
	httpClient := http.Client{
		Timeout: time.Second * 10,
	}

	req, err := http.NewRequest(http.MethodGet, *apiServerAddr+"/v1/config", nil)
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

	config := config.Config{}
	err = json.Unmarshal(body, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil

}

func main() {
	flag.Parse()
	if *prometheusActive {
		prometheus.MustRegister(promTemperature)
		prometheus.MustRegister(promWeight)
	}
	config := getConfig()

	http.HandleFunc("/scale", scaleHandler)
	http.HandleFunc("/temperature", temperatureHandler)
	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/bhive", func(res http.ResponseWriter, req *http.Request) {
		http.ServeFile(res, req, *httpServerHiveFile)
	})

	relaySwitches := []relay.Switcher{&relay.Switch{Gpio: 16}}
	relay := relay.RelayModule{}
	err := relay.Initialize(relaySwitches)
	if err != nil {
		logger.Debug("", fmt.Sprintf("bbox relay problems: %s", err))
	}

	scheduler := scheduler.Schedule{Schedule: "*/5 * * * *", RelayModule: relay}
	c := make(chan bool)
	go scheduler.Start(c)

	go bBuffer.FlushSchedule(apiServerAddr, "token", *flushInterval)

	log.Fatal(http.ListenAndServe(":"+*httpServerPort, nil))
}
