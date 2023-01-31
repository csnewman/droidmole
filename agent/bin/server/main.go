package main

import (
	"github.com/csnewman/droidmole/agent/server"
	"github.com/csnewman/droidmole/agent/server/emulator"
)

func main() {
	go emulator.Run()

	s := server.New()
	s.Start()
}
