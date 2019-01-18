package net

import (
	"github.com/temprory/log"
)

var (
	logDebug = log.Debug
)

func SetDebugLogger(logger func(format string, v ...interface{})) {
	logDebug = logger
}
