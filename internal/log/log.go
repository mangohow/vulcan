package log

import (
	"fmt"
	"github.com/gookit/color"
	"os"
	"strings"
)

var debug = true

func Debugf(format string, args ...interface{}) {
	if !debug {
		return
	}
	if !strings.HasSuffix(format, "\n") {
		format += "\n"
	}
	fmt.Printf(format, args...)
}

func Logf(format string, args ...interface{}) {
	if !strings.HasSuffix(format, "\n") {
		format += "\n"
	}
	fmt.Printf(format, args...)
}

func Infof(format string, args ...interface{}) {
	if !strings.HasSuffix(format, "\n") {
		format += "\n"
	}
	color.Green.Printf(format, args...)
}

func Errorf(format string, args ...interface{}) {
	if !strings.HasSuffix(format, "\n") {
		format += "\n"
	}
	color.Red.Printf(format, args...)
}

func Fatalf(format string, args ...interface{}) {
	if !strings.HasSuffix(format, "\n") {
		format += "\n"
	}
	color.Red.Printf(format, args...)
	os.Exit(1)
}
