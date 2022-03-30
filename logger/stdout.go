package logger

import (
	"fmt"
	"os"
	"strconv"
)

var debug bool

func init() {
	debugEnv, err := strconv.ParseBool(os.Getenv("debug"))
	if err != nil {
		debug = false
	}
	debug = debugEnv
}

func Info(format string, a ...interface{}) {
	if !debug {
		return
	}

	if a == nil {
		fmt.Println(format)
	} else {
		fmt.Printf(format+"\n", a)
	}
}

func Error(a string) {
	_, err := fmt.Fprintln(os.Stderr, fmt.Sprintf("ERROR. %s\n", a))
	if err != nil {
		fmt.Println("???")
	}
}

func Enabled() bool {
	return debug
}
