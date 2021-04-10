package relay

import (
	"testing"
	//"github.com/google/go-cmp/cmp"
	// if diff := cmp.Diff(tempSlice, te); diff != "" {
)

type SwitchMock struct {
	Gpio  int
	State bool
}

func (h *SwitchMock) Off() error {
	h.State = false
	return nil
}

func (h *SwitchMock) On() error {
	h.State = true
	return nil
}

func (h *SwitchMock) GetState() bool {
	return h.State
}

func TestEmptyRelay(t *testing.T) {
	relay := RelayModule{}
	emptySlice := []Switcher{}
	err := relay.Initialize(emptySlice)
	if err != nil {
		t.Fatalf("relay couldn't be initialized")
	}
	done, err := relay.ActivateNextBHive()
	if done != true {
		t.Errorf("Empty GPIOs for relay didn't work")
	}
}

func TestSingleRelay(t *testing.T) {
	r := RelayModule{}
	relaySwitches := []Switcher{&SwitchMock{Gpio: 16}}
	err := r.Initialize(relaySwitches)
	if err != nil {
		t.Fatalf("relay couldn't be initialized: %s", err)
		return
	}
	done, err := r.ActivateNextBHive()
	if done == true {
		t.Errorf("One switch shouldn't return success here.")
	}
	if r.Switches[0].GetState() == false {
		t.Errorf("Switch should be on!")
	}
	done, err = r.ActivateNextBHive()
	if done == false {
		t.Errorf("We should be done by now!")
	}
	if r.Switches[0].GetState() == true {
		t.Errorf("Switch should be off!")
	}
}

func TestThreeRelays(t *testing.T) {
	r := RelayModule{}
	relaySwitches := []Switcher{&SwitchMock{Gpio: 16}, &SwitchMock{Gpio: 15}, &SwitchMock{Gpio: 14}}
	err := r.Initialize(relaySwitches)
	if err != nil {
		t.Fatalf("relay couldn't be initialized: %s", err)
		return
	}
	for i := range []int{1, 2, 3} {
		if r.Switches[i].GetState() == true {
			t.Errorf("Switch is supposed to be turned off!")
		}
	}

	done, err := r.ActivateNextBHive()
	if done == true {
		t.Errorf("This shouldn't return success here.")
	}
	if r.Switches[0].GetState() == false {
		t.Errorf("Switch should be on!")
	}

	done, err = r.ActivateNextBHive()
	if done == true {
		t.Errorf("This shouldn't return success here.")
	}
	if r.Switches[0].GetState() != false {
		t.Errorf("Switch should be off!")
	}
	if r.Switches[1].GetState() != true {
		t.Errorf("Switch should be on!")
	}

	done, err = r.ActivateNextBHive()
	if done == true {
		t.Errorf("This shouldn't return success here.")
	}
	if r.Switches[1].GetState() != false {
		t.Errorf("Switch should be off!")
	}
	if r.Switches[2].GetState() != true {
		t.Errorf("Switch should be on!")
	}

	done, err = r.ActivateNextBHive()
	if done != true {
		t.Errorf("This shouldn't return false here.")
	}
	if r.Switches[0].GetState() != false {
		t.Errorf("Switch should be off!")
	}
	if r.Switches[1].GetState() != false {
		t.Errorf("Switch should be off!")
	}
	if r.Switches[2].GetState() != false {
		t.Errorf("Switch should be off!")
	}

}
