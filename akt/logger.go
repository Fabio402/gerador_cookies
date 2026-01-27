package akt

import (
	"os"
	"fmt"
)

func ConsoleLog(message ...interface{}) {
	if os.Getenv("isDebug") == "true" {
		fmt.Println(message...)
	}
}
