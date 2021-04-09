package relais

import (
  "github.com/stianeikeland/go-rpio/v4"
)

func OpenAllRelais(gpios []int) error {
  err := rpio.Open()
  if err != nil {
    return err
  }
  defer rpio.Close()

  for _, gpio := range gpios {
    pin := rpio.Pin(gpio)
    pin.Output()  // Output mode
    pin.High()
    // pin.Low()
  }
  return nil
}
