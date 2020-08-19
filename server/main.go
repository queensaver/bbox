package main

import (
	"flag"
	"log"
  "net/http"
	"github.com/wogri/bbox/thingspeak_client"
)

var httpServerPort = flag.Int("http_server_port", "8333", "HTTP server port")
var debug = flag.Bool("debug", false, "debug mode")
var thingspeakKey = flag.String("thingspeak_api_key", "48PCU5CAQ0BSP4CL", "API key for Thingspeak")
var thingspeakActive = flag.Bool("thingspeak", false, "Activate thingspeak API if set to true")

func main() {
	flag.Parse()

	thing := thingspeak_client.NewChannelWriter(*thingspeakKey)
	thing.AddField(1, medianWeight)
	if *thingspeakActive {
		if *debug {
			log.Println("uploading data to Thingspeak...")
		}
		_, err = thing.Update()
		if err != nil {
			log.Println("ThingSpeak error:", err)
		}
	}
}
