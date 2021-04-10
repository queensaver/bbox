package scheduler

import (
	"fmt"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/wogri/bbox/packages/logger"
	"github.com/wogri/bbox/server/relay"
)

type Schedule struct {
	Schedule    string
	RelayModule relay.RelayModule
	cron        *cron.Cron
}

func (s *Schedule) runSchedule() {
	logger.Debug("none", "runSchedule started")
	for {
		done, err := s.RelayModule.ActivateNextBHive()
		if err != nil {
			logger.Debug("none", fmt.Sprintf("%s", err))
			continue
		}
		if done {
			break
		}
		logger.Debug("none", "runSchedule sleeping")
		time.Sleep(2 * time.Minute)
	}
	logger.Debug("none", "runSchedule done")
}

func (s *Schedule) Start(killswitch chan bool) {
	s.cron = cron.New()
	s.cron.AddFunc(s.Schedule, s.runSchedule)
	s.runSchedule() // TODO: Remove me when we have a real server
	s.cron.Start()
	<-killswitch
	s.cron.Stop()
}
