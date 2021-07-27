.PHONY: all
all: 
	env GOOS=linux GOARCH=arm GOARM=6 go build -ldflags="-s -w" -v

copy: all
	scp bla pi@192.168.1.156: && ssh pi@192.168.1.156 ./bla
	
