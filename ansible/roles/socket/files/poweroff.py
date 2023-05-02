#!/usr/bin/python3

import RPi.GPIO as GPIO           # import RPi.GPIO module
GPIO.setmode(GPIO.BCM)            # choose BCM or BOARD
n = 5
GPIO.setup(n, GPIO.OUT) # set a port/pin as an output
GPIO.output(n, 0)       # set port/pin value to 0/GPIO.LOW/False
