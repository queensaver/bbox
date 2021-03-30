/* This is a test application, not used in the actual package. */
package main

import (
  "github.com/stianeikeland/go-rpio/v4"
  "log"
  "flag"
  "time"
)

var duration = flag.Int("s", 20, "seconds for when the relais turns on")
var gpioPin = flag.Int("gpio-pin", 17, "GPIO Pin")


func main() {
  flag.Parse()

  err := rpio.Open()
  if err != nil {
    log.Fatal(err)
  }
  defer rpio.Close()
  pin := rpio.Pin(*gpioPin)
  pin.Output()       // Output mode
  log.Println("Low")
  pin.Low()
  time.Sleep(time.Duration(*duration) * time.Second)
  log.Println("High")
  pin.High()         // Set pin High
}
