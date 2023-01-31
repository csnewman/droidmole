package main

import (
	"github.com/csnewman/droidmole/agent/server"
)

func main() {
	s := server.New()
	s.Start()
}
