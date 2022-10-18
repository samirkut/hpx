package main

import (
	"log"
)

var debugf = func(format string, args ...any) {}

func verboseDebug(format string, args ...any) {
	log.Printf(format, args...)
}
