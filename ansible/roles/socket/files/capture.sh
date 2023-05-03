#!/bin/bash
set -e
python3 /home/pi/hw.py --debug scan do light 1
sleep 1
# D=$(date +%s)
FILENAME=/home/pi/bOS/scan.jpg
D=$(date +%s)
libcamera-jpeg --brightness=0.1 -o $FILENAME
python3 /home/pi/hw.py --debug scan do light 0
mac=$(ip link show dev wlan0 | grep link/ether | awk '{ print $2 }' | tr -d \:)
curl -v -F bhiveId=$mac -F epoch=$D -F scan=@$FILENAME http://localhost/varroa 
python3 /home/pi/hw.py --debug scan do move 100 6
