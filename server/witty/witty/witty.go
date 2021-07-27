package witty

import (
	"periph.io/x/conn/v3/i2c"
  host "periph.io/x/host/v3"
	"periph.io/x/conn/v3/i2c/i2creg"
  "fmt"
	"time"
)

func dec2bcd(num int) byte {
	return byte(num/10*16 + num%10)
}

func write(dev *i2c.Dev, val []byte) error {
  read := make([]byte, 1)
  err := dev.Tx(val, read)
  if err != nil {
		return err
	}
	fmt.Printf("%v\n", read)
  return nil
}

func StartAt(t time.Time) error {
  _, err := host.Init()
	// _, err := driverreg.Init()
	if err != nil {
		return err
	}
	// Use i2creg I²C bus registry to find the first available I²C bus.
	b, err := i2creg.Open("")
	if err != nil {
		return err
	}
	defer b.Close()

	// Dev is a valid conn.Conn.
	d := &i2c.Dev{Addr: 0x68, Bus: b}

  // initialize the setup - I think.
  err = write(d, []byte{0x0E, 0x07})
  if err != nil {
    return err
  }


  // 10 seconds
  err = write(d, []byte{0x07, dec2bcd(10)})
  if err != nil {
    return err
  }

  return nil
}

/* Based on this shellscript

Usage: i2cset [-f] [-y] [-m MASK] [-r] [-a] I2CBUS CHIP-ADDRESS DATA-ADDRESS [VALUE] ... [MODE]
  I2CBUS is an integer or an I2C bus name
  ADDRESS is an integer (0x03 - 0x77, or 0x00 - 0x7f if -a is given)
  MODE is one of:
    c (byte, no value)
    b (byte data, default)
    w (word data)
    i (I2C block data)
    s (SMBus block data)
    Append p for SMBus PEC


I2C_RTC_ADDRESS=0x68

i2c_write()
{
  local retry=0
  if [ $# -gt 4 ] ; then
    retry=$5
  fi
  i2cset -y $1 $2 $3 $4
  local result=$(i2c_read $1 $2 $3)
  if [ "$result" != $(dec2hex "$4") ] ; then
    retry=$(( $retry + 1 ))
    if [ $retry -eq 4 ] ; then
      log "I2C write $1 $2 $3 $4 failed (result=$result), and no more retry."
    else
      sleep 1
      log2file "I2C write $1 $2 $3 $4 failed (result=$result), retrying $retry ..."
      i2c_write $1 $2 $3 $4 $retry
    fi
  fi
}


set_startup_time()
{
  i2c_write 0x01 $I2C_RTC_ADDRESS 0x0E 0x07
  if [ $4 == '??' ]; then
    sec='128'
  else
    sec=$(dec2bcd $4)
  fi
  i2c_write 0x01 $I2C_RTC_ADDRESS 0x07 $sec
  if [ $3 == '??' ]; then
    min='128'
  else
    min=$(dec2bcd $3)
  fi
  i2c_write 0x01 $I2C_RTC_ADDRESS 0x08 $min
  if [ $2 == '??' ]; then
    hour='128'
  else
    hour=$(dec2bcd $2)
  fi
  i2c_write 0x01 $I2C_RTC_ADDRESS 0x09 $hour
  if [ $1 == '??' ]; then
    date='128'
  else
    date=$(dec2bcd $1)
  fi
  i2c_write 0x01 $I2C_RTC_ADDRESS 0x0A $date
}

*/
