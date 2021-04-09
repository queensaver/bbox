package relay

import (
	"github.com/stianeikeland/go-rpio/v4"
)

type RelayModule struct {
	Switches      []Switcher
	currentSwitch int
}

type Switch struct {
	Gpio  int
	State bool
}

type Switcher interface {
	Off() error
	On() error
}

// turns off power from the current switch and turns on the next one.
// Returns:
//   true if the last switch was reached (and it will be turned off). This means we're done with this measurement cycle for all bhives.
func (r *RelayModule) ActivateNextBHive() (bool, error) {
	if len(r.Switches) == 0 {
		return true, nil
	}
	if r.currentSwitch != -1 {
		r.Switches[r.currentSwitch].Off()
	}
	r.currentSwitch++

	// we are done with the last switch and we're turning off the last bhive and claim victory.
	if r.currentSwitch == len(r.Switches) {
		r.currentSwitch = -1
		return true, nil
	}
	r.Switches[r.currentSwitch].On()
	return false, nil

}

// Closes the switch - current can go through
func (s *Switch) On() error {
	err := rpio.Open()
	if err != nil {
		return err
	}
	defer rpio.Close()

	pin := rpio.Pin(s.Gpio)
	pin.Output()
	pin.Low()
	s.State = true
	return nil
}

// Opens the switch - no current can go through
func (s *Switch) Off() error {
	err := rpio.Open()
	if err != nil {
		return err
	}
	defer rpio.Close()

	pin := rpio.Pin(s.Gpio)
	pin.Output()
	pin.High()
	s.State = false
	return nil
}

// Turns all switches off during initalization
// Arguments:
//		gpios: GPIO numbers on raspberrry pi that are used for the relay module
// Returns:
// 		Error
func (r *RelayModule) Initialize(Switches []Switcher) error {
	r.currentSwitch = -1
	for _, s := range Switches {
		r.Switches = append(r.Switches, s)
	}
	for _, s := range r.Switches {
		err := s.Off()
		if err != nil {
			return err
		}
	}
	return nil
}
