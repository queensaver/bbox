package main

import (
  "witty"
  "time"
  "fmt"
)

func main() {
  t := time.Now()
  err := witty.StartAt(t.Add(time.Hour * 1))
  fmt.Println(err)
}
