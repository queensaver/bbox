/* This is a test application, not used in the actual package. */
package main

import (
  "github.com/stianeikeland/go-rpio/v4"
  "log"
  "fmt"
  "time"
)

func main() {
  err := rpio.Open()
  if err != nil {
    log.Fatal(err)
  }
  defer rpio.Close()
  pin := rpio.Pin(17)
  pin.Output()       // Output mode
  fmt.Println("High")
  pin.High()         // Set pin High
  time.Sleep(1 * time.Second)
  fmt.Println("Low")
  pin.Low()
}
