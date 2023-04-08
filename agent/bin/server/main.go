package main

import (
	"github.com/csnewman/droidmole/agent/server"
	"github.com/csnewman/droidmole/agent/server/adb"
	"go.uber.org/zap"
)

func main() {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()
	sugar := logger.Sugar()

	sugar.Info("DroidMole Agent")
	sugar.Info("Configuring")

	adbFactory := adb.NewRawConnectionFactory()
	adb := adb.New(sugar, adbFactory)
	server := server.New(sugar, adb)
	
	server.Start()
}
