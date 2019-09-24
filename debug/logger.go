package debug

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

func Info(a string) {
	if debug {
		fmt.Print(a + "\n")
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
