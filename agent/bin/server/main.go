package main

import (
	"github.com/csnewman/droidmole/agent/server"
	"github.com/csnewman/droidmole/agent/server/adb"
	"github.com/csnewman/droidmole/agent/util/di"
	"log"
)

func main() {
	c, err := di.New(
		di.Provider(server.New),
		di.Provider(adb.New),
	)
	if err != nil {
		log.Fatal(err)
	}

	var s *server.Server
	err = c.Get(&s)
	if err != nil {
		log.Fatal(err)
	}

	s.Start()
}
