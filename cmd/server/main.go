package main

import (
	"os"

	zl "github.com/alexferl/zerohttp/log"

	"github.com/alexferl/zerohttp-example/app"
	"github.com/alexferl/zerohttp-example/config"
)

func main() {
	zl.SetGlobalLogger(zl.NewDefaultLogger())

	cfg, err := config.Load()
	if err != nil {
		zl.GetGlobalLogger().Fatal("Failed to load config", zl.E(err))
	}

	application, err := app.New(cfg)
	if err != nil {
		zl.GetGlobalLogger().Fatal("Failed to create app", zl.E(err))
		os.Exit(1)
	}

	if err := application.Start(); err != nil {
		zl.GetGlobalLogger().Fatal("Application error", zl.E(err))
		os.Exit(1)
	}
}
