package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/google/go-github/v34/github"
)

var releaseTrain = flag.String("release_train", "stable", "GitHub release train name")
var updateCheckInterval = flag.Int("update_check_interval", 60*60*24, "how often to check for updates")

func getReleaseID(org, repo, train string) (int64, error) {
	client := github.NewClient(nil)
	if client == nil {
		return 0, errors.New("nil client")
	}
	release, _, err := client.Repositories.GetReleaseByTag(context.Background(), org, repo, train)
	if err != nil {
		return 0, err
	}
	return release.GetID(), nil
}

func waitRandomTime(t int) {
	fmt.Println("waiting between 0 and", t, "seconds")
	time.Sleep(time.Duration(rand.Intn(t)) * time.Second)
}
func checkRelease(org, repo, train, binary, filename string, old_id int64) (int64, error) {
	var id int64
	var err error
	bo := backoff.NewExponentialBackOff()
	bo.MaxInterval = 60 * time.Minute
	exp := 0
	for {
		id, err = getReleaseID(org, repo, train)
		exp += 1
		if exp > 10 {
			exp = 10
		}
		if err != nil {
			fmt.Println(err)
			waitRandomTime(10 * exp)
		} else {
			break
		}
	}
	if id != old_id {
		err := backoff.Retry(func() error {
			fmt.Println("downloading the latest release")
			err := downloadRelease(fmt.Sprintf("/home/pi/bOS/%s", filename), fmt.Sprintf("https://github.com/%s/%s/releases/download/%s/%s", org, repo, train, binary))
			if err != nil {
				fmt.Println("error downloading:", err)
			}
			return err
		}, bo)
		if err != nil {
			return old_id, err
		}
		return id, nil
	}
	return old_id, nil
}

// DownloadFile will download a url to a local file. It's efficient because it will
// write as it downloads and not load the whole file into memory.
func downloadRelease(filepath string, url string) error {

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	return err
}

func main() {
	flag.Parse()
	var err error

	go func() {
		var id, old_id int64
		old_id = 0
		for {
			id, err = checkRelease("btelemetry", "bhive", *releaseTrain, "bhive", "bhive", old_id)
			if err != nil {
				fmt.Println("error restarting server:", err)
			}
			if old_id != id {
				old_id = id
			}
			waitRandomTime(*updateCheckInterval)
		}
	}()

	go func() {
		var id, old_id int64
		old_id = 0
		for {
			id, err = checkRelease("btelemetry", "bbox", *releaseTrain, "server", "server.download", old_id)
			if err != nil {
				fmt.Println(err)
			}
			if old_id != id {
				cmd := exec.Command("/usr/bin/systemctl", "stop", "server")
				err = cmd.Run()
				if err != nil {
					fmt.Println("error restarting server:", err)
					continue
				}
				err = os.Rename("home/pi/bOS/server.download", "home/pi/bOS/server")
				if err != nil {
					fmt.Println("error renaming file:", err)
					continue
				}
				cmd = exec.Command("/usr/bin/systemctl", "start", "server")
				err = cmd.Run()
				if err != nil {
					fmt.Println("error restarting server:", err)
					continue
				}
				old_id = id
			}
			waitRandomTime(*updateCheckInterval)
		}
	}()
	exitSignal := make(chan os.Signal)
	signal.Notify(exitSignal, syscall.SIGINT, syscall.SIGTERM)
	<-exitSignal

}
