package main

import (
	"encoding/json"
	"flag"
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

func thingSpeakTemperatureUpdate(temperature float64) error {
	thing := thingspeak_client.NewChannelWriter(*thingspeakKey)
	err := thing.SendTemperature(temperature)
	return err
}

func thingSpeakWeightUpdate(weight float64) error {
	thing := thingspeak_client.NewChannelWriter(*thingspeakKey)
	err := thing.SendWeight(weight)
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
}

func main() {
	flag.Parse()

	http.HandleFunc("/scale", scaleHandler)
	http.HandleFunc("/temperature", temperatureHandler)
	log.Fatal(http.ListenAndServe(":"+*httpServerPort, nil))
}
