package main

import (
	"github.com/csnewman/droidmole/agent/server"
	"github.com/csnewman/droidmole/agent/server/adb"
	"github.com/csnewman/droidmole/agent/util/di"
	"go.uber.org/zap"
)

func main() {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()
	sugar := logger.Sugar()

	sugar.Info("DroidMole Agent")
	sugar.Info("Configuring")

	c, err := di.New(
		di.Value(sugar),
		di.Provider(server.New),
		di.Provider(adb.New),
	)
	if err != nil {
		sugar.Fatal(err)
	}

	var s *server.Server
	err = c.Get(&s)
	if err != nil {
		sugar.Fatal(err)
	}

	s.Start()
}
