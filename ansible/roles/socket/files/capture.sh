#!/bin/bash
python3 /home/pi/hw.py --debug scan do light 1
sleep 1
# D=$(date +%s)
D=/home/pi/scan
libcamera-jpeg --brightness=0.1 -o bOS/$D.jpg
python3 /home/pi/hw.py --debug scan do light 0
mac=$(ip link show dev eth0 | grep link/ether | awk '{ print $2 }' | tr -d \:)
curl -v -F bhiveId=$mac -F epoch=$D -F scan=@bOS/$D.jpg http://localhost/varroa && \
  rm bOS/$D.jpg
python3 /home/pi/hw.py --debug scan do move 100 6
