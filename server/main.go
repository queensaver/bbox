package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/wogri/bbox/packages/buffer"
	"github.com/wogri/bbox/packages/logger"
	"github.com/wogri/bbox/packages/scale"
	"github.com/wogri/bbox/packages/temperature"
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

func main() {
	flag.Parse()
	if *prometheusActive {
		prometheus.MustRegister(promTemperature)
		prometheus.MustRegister(promWeight)
	}

	http.HandleFunc("/scale", scaleHandler)
	http.HandleFunc("/temperature", temperatureHandler)
	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/bhive", func(res http.ResponseWriter, req *http.Request) {
		http.ServeFile(res, req, *httpServerHiveFile)
	})
	go bBuffer.FlushSchedule(apiServerAddr, "token", *flushInterval)
	log.Fatal(http.ListenAndServe(":"+*httpServerPort, nil))
}
