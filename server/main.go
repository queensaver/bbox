package main

import (
	"encoding/json"
	"flag"
	"github.com/wogri/bbox/thingspeak_client"
	scale "github.com/wogri/bbox/structs"
	"log"
	"net/http"
)

var httpServerPort = flag.String("http_server_port", "8333", "HTTP server port")
var debug = flag.Bool("debug", false, "debug mode")
var thingspeakKey = flag.String("thingspeak_api_key", "48PCU5CAQ0BSP4CL", "API key for Thingspeak")
var thingspeakActive = flag.Bool("thingspeak", false, "Activate thingspeak API if set to true")

func thingSpeakUpdate(weight float64) error {
	if *thingspeakActive {
		thing := thingspeak_client.NewChannelWriter(*thingspeakKey)
		thing.AddField(1, weight)
		if *debug {
			log.Println("uploading data to Thingspeak...")
		}
		_, err := thing.Update()
		if err != nil {
			return err
		}
	}
	return nil

}

func scaleHandler(w http.ResponseWriter, req *http.Request) {
	decoder := json.NewDecoder(req.Body)
	var s scale.Scale
	err := decoder.Decode(&s)
	if err != nil {
		log.Println(err)
	}
	out, err := s.String()
	if *debug {
		log.Println(string(out))
	}
}

func main() {
	flag.Parse()

	http.HandleFunc("/scale", scaleHandler)
	log.Fatal(http.ListenAndServe(":"+*httpServerPort, nil))
}
