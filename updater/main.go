package main

import (
	"context"
	"errors"
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
func checkRelease(org, repo, train, binary string, old_id int64) (int64, error) {
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
			err := downloadRelease(fmt.Sprintf("/home/pi/bOS/%s", binary), fmt.Sprintf("https://github.com/%s/%s/releases/download/%s/%s", org, repo, train, binary))
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
	var err error

	go func() {
		var id, old_id int64
		old_id = 0
		for {
			id, err = checkRelease("wogri", "bhive", "stable", "bhive", old_id)
			if err != nil {
				fmt.Println("error restarting server:", err)
			}
			if old_id != id {
				old_id = id
			}
			waitRandomTime(24 * 60 * 60)
		}
	}()

	go func() {
		var id, old_id int64
		old_id = 0
		for {
			id, err = checkRelease("wogri", "bbox", "stable", "server", old_id)
			if err != nil {
				fmt.Println(err)
			}
			if old_id != id {
				cmd := exec.Command("/usr/bin/systemctl", "restart", "server")
				err = cmd.Run()
				if err != nil {
					fmt.Println("error restarting server:", err)
				}
				old_id = id
			}
			waitRandomTime(24 * 60 * 60)
		}
	}()
	exitSignal := make(chan os.Signal)
	signal.Notify(exitSignal, syscall.SIGINT, syscall.SIGTERM)
	<-exitSignal

}
