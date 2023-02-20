package main

import "github.com/csnewman/droidmole/demo/backend/server"

func main() {
	s := server.New()
	s.Start()
}
