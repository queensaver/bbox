package main

import (
	"encoding/json"
	"flag"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/wogri/bbox/structs/scale"
	"github.com/wogri/bbox/structs/temperature"
	"github.com/wogri/bbox/thingspeak_client"
	"log"
	"net/http"
)

var httpServerPort = flag.String("http_server_port", "8333", "HTTP server port")
var debug = flag.Bool("debug", false, "debug mode")
var thingspeakKey = flag.String("thingspeak_api_key", "48PCU5CAQ0BSP4CL", "API key for Thingspeak")
var thingspeakActive = flag.Bool("thingspeak", false, "Activate thingspeak API if set to true")
var prometheusActive = flag.Bool("prometheus", false, "Activate Prometheus exporter")

var thing = thingspeak_client.NewChannelWriter(*thingspeakKey)

var (
	promTemperature = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "bhive_temperature",
		Help: "Temperature of the bHive",
	},
		[]string{"BBoxID", "SensorID"},
	)
	promWeight = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "bhive_weight",
		Help: "Weight of the bHive",
	},
		[]string{"BBoxID"},
	)
)

func thingSpeakTemperatureUpdate(temperature float64) error {
	thing.SetTemperature(temperature)
	_, err := thing.Update()
	return err
}

func thingSpeakWeightUpdate(weight float64) error {
	thing.SetWeight(weight)
	_, err := thing.Update()
	return err
}

func temperatureHandler(w http.ResponseWriter, req *http.Request) {
	decoder := json.NewDecoder(req.Body)
	var t temperature.Temperature
	err := decoder.Decode(&t)
	if err != nil {
		log.Println(err)
	}
	if *debug {
		out, _ := t.String()
		log.Println(string(out))
	}
	if *thingspeakActive {
		err = thingSpeakTemperatureUpdate(t.Temperature)
		if err != nil {
			log.Println(err)
		}
	}
	if *prometheusActive {
		promTemperature.WithLabelValues(t.BBoxID, t.SensorID).Set(t.Temperature)
	}
}

func scaleHandler(w http.ResponseWriter, req *http.Request) {
	decoder := json.NewDecoder(req.Body)
	var s scale.Scale
	err := decoder.Decode(&s)
	if err != nil {
		log.Println(err)
	}
	if *debug {
		out, _ := s.String()
		log.Println(string(out))
	}
	if *thingspeakActive {
		err = thingSpeakWeightUpdate(s.Weight)
		if err != nil {
			log.Println(err)
		}
	}
	if *prometheusActive {
		promWeight.WithLabelValues(s.BBoxID).Set(s.Weight)
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
	log.Fatal(http.ListenAndServe(":"+*httpServerPort, nil))
}
