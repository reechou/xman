package main

import (
	"os"

	. "youzan/youzan_controller/controller"
)

func main() {
	argNum := len(os.Args)
	if argNum >= 2 {
		ConfigPath = os.Args[1]
	}

	InitConfig()
	Logic.Run()
}
