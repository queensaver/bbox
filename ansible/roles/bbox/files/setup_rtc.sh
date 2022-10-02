#!/bin/sh
/usr/sbin/i2cset -f -y 1 0x68 0x10 0xA6
/bin/echo ds1307 0x68 > /sys/class/i2c-adapter/i2c-1/new_device
modprobe rtc_ds1307
sleep 3
