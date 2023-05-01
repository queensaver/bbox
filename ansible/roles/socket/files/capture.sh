#!/bin/bash
python3 /home/pi/hw.py --debug scan do light 1
sleep 1
# D=$(date +%s)
D=/home/pi/scan
libcamera-jpeg --brightness=0.1 -o $D.jpg
python3 /home/pi/hw.py --debug scan do light 0
mac=$(ip link show dev vnet15 | grep link/ether | awk '{ print $2 }' | tr -d \:)
curl -v -F bhiveId=$mac -F epoch=$D -F scan=@$D.jpg http://localhost/varroa
python3 /home/pi/hw.py --debug scan do move 100 6
