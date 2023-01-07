package main

import (
	"droidmole/server/connection"
	"log"
	"os"
	"time"
)

func main() {
	_, err := connection.New(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}

	for {
		time.Sleep(time.Duration(1) * time.Second)
	}
}
