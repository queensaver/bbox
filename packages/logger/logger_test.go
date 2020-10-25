package logger

import (
  "testing"
  // "github.com/wogri/bbox/packages/logger"
)

type TestError struct{
  message string
}

func (t TestError) Error() string {
	return "This is a Test Error"
}

func TestLogger(t *testing.T) {
  Error("1.2.3.5", TestError{})
}

