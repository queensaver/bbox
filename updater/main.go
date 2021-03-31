package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/google/go-github/v34/github"
)

func getReleaseID() (int64, error) {
	client := github.NewClient(nil)
	if client == nil {
		return 0, errors.New("nil client")
	}
	release, _, err := client.Repositories.GetReleaseByTag(context.Background(), "wogri", "bbox", "stable")
	if err != nil {
		return 0, err
	}
	return release.GetID(), nil
}

func waitRandomTime(t int) {
	fmt.Println("waiting between 0 and", t, "seconds")
	time.Sleep(time.Duration(rand.Intn(t)) * time.Second)
}
func checkRelease() int64 {
	var id int64
	var err error
	exp := 0
	for {
		id, err = getReleaseID()
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
	return id
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
	var id, old_id int64
	old_id = 0
	bo := backoff.NewExponentialBackOff()
	bo.MaxInterval = 60 * time.Minute

	for {
		id = checkRelease()
		fmt.Println("release id:", id)
		if id != old_id {
			err := backoff.Retry(func() error {
				return downloadRelease("/tmp/server", "https://github.com/wogri/bbox/releases/download/stable/server")
			}, bo)
			if err != nil {
				fmt.Println(err)
			}
			old_id = id
		}
		waitRandomTime(60)
	}
}
