package scheduler

import (
	"fmt"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/queensaver/packages/logger"
	"github.com/queensaver/bbox/server/relay"
)

type Schedule struct {
	Schedule    string
	RelayModule relay.RelayModule
  Token       string
	cron        *cron.Cron
}

func (s *Schedule) runSchedule() {
  /* TODO: download config with:
	bConfig, err = config.Get(*apiServerAddr + "/v1/config", token)
	// TODO: this needs to be downloaded before every scheduler run
	if err != nil {
		log.Fatal(err)
	}
  Then update all the relevant datapoints.
  */

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
