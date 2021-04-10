package scheduler

import (
	"github.com/robfig/cron/v3"
	"github.com/wogri/bbox/server/relay"
	"time"
	"fmt"
)

type Schedule struct {
	relayModule relay.RelayModule
	Cron *cron.Cron
}

func (s *Schedule) runSchedule() {
  for {
    done, err := s.relayModule.ActivateNextBHive()
    if err != nil {
      fmt.Println(err)
    }
    if done {
      break
    }
    time.Sleep(2 * time.Minute)
  }
}

func (s *Schedule) InitCron(schedule string) {
  s.Cron = cron.New()
  s.Cron.AddFunc(schedule, s.runSchedule)
  s.Cron.Start()
}
