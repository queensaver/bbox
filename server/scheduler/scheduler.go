package scheduler

import (
	"fmt"
	"log"
	"os/exec"
	"time"

	"github.com/queensaver/bbox/server/relay"
	"github.com/queensaver/packages/logger"
	"github.com/robfig/cron/v3"
)

type Schedule struct {
	Schedule    string
	RelayModule relay.RelayModule
	Token       string
	Local       bool
	cron        *cron.Cron
	HiveBinary  string
}

// This function could have some more paramters like the
func (s *Schedule) runLocally() {
  log.Println("starting bhive client locally")
	cmd := exec.Command("/usr/bin/systemctl", "restart", "bhive.service")
	err := cmd.Run()
	if err != nil {
		log.Println("error restarting server:", err)
	}
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
	if s.Local {
		s.cron.AddFunc(s.Schedule, s.runLocally)
	} else {
		s.cron.AddFunc(s.Schedule, s.runSchedule)
	}
	s.runSchedule() // TODO: Remove me when we run in complete production - this just triggers the run immediately for convenience.gw
	s.cron.Start()
	<-killswitch
	s.cron.Stop()
}
