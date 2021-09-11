package scheduler

import (
	"fmt"
	"math"
	"os/exec"
	"sync"
	"time"

	"github.com/queensaver/bbox/server/relay"
	"github.com/queensaver/bbox/server/witty"
	"github.com/queensaver/packages/logger"
	"github.com/robfig/cron/v3"
)

type Schedule struct {
	Schedule     string
	RelayModule  relay.RelayModule
	Token        string
	Local        bool
	WittyPi      bool
	cron         *cron.Cron
	localRunLock bool // Set to true when the scheduler is running locally and waitig for results from bhive code.
	mu           sync.Mutex
}

func (s *Schedule) LocalRunBusy() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.localRunLock
}

func (s *Schedule) UnSetLocalRunLock() {
	s.mu.Lock()
	s.localRunLock = false
	s.mu.Unlock()
}

func (s *Schedule) SetLocalRunLock() {
	s.mu.Lock()
	s.localRunLock = true
	s.mu.Unlock()
}

// This function could have some more paramters like the
func (s *Schedule) runLocally() {
	logger.Debug("none", "starting bhive client locally")
	s.SetLocalRunLock()
	cmd := exec.Command("/usr/bin/systemctl", "restart", "bhive.service")
	err := cmd.Run()
	if err != nil {
		logger.Error("error restarting server:", err)
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

// Returns true if a shutdown is useful, false if it doesn't make sense (that might be because the next scheduled startup is already in the next 120 seconds)
func (s *Schedule) Shutdown() bool {
	entries := s.cron.Entries()
	next := entries[0].Next
	logger.Debug("the next time witty pi will turn on the machine: ", fmt.Sprintf("%+v", next))
	if math.Abs(time.Until(next).Seconds()) < 120 {
		fmt.Println("not shutting down the raspberry, next startup time is in under 120 seconds.")
		return false
	}
	witty.StartAt(next)
	cmd := exec.Command("/usr/sbin/shutdown", "-h", "now")
	err := cmd.Run()
	if err != nil {
		fmt.Println(err)
		return false
	}
	return true
}

func (s *Schedule) Start(killswitch chan bool) {
	s.cron = cron.New()
	if s.Local {
		s.cron.AddFunc(s.Schedule, s.runLocally)
		s.cron.Start()
		s.runLocally() // TODO: Remove me when we run in complete production - this just triggers the run immediately for convenience.gw
	} else {
		s.cron.AddFunc(s.Schedule, s.runSchedule)
		s.cron.Start()
		s.runSchedule() // TODO: Remove me when we run in complete production - this just triggers the run immediately for convenience.gw
	}
	<-killswitch
	s.cron.Stop()
}
