package main

import (
  "bla/witty"
  "time"
  "fmt"
)

func main() {
  t := time.Now()
  err := witty.StartAt(t)
  fmt.Println(err)
}
