package logging

import (
    "github.com/sirupsen/logrus"
)

var (
    log *logrus.Logger
)

func init() {
    log = logrus.New()
    log.Formatter = &logrus.JSONFormatter{}

    log.SetReportCaller(true)
}

func Info(format string, v ...interface{}) {
    log.Infof(format, v...)
}

func Warn(format string, v ...interface{}) {
    log.Warnf(format, v...)
}

func Error(format string, v ...interface{}) {
    log.Errorf(format, v...)
}

var (

    // ConfigError ...
    ConfigError = "%v type=config.error"

    // HTTPError ...
    HTTPError = "%v type=http.error"

    // HTTPWarn ...
    HTTPWarn = "%v type=http.warn"

    // HTTPInfo ...
    HTTPInfo = "%v type=http.info"
)
