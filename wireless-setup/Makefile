wireless-setup: setup.go
	go build

arm:
	env GOOS=linux GOARCH=arm GOARM=6 go build -ldflags="-s -w" -v
