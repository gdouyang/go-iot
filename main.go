package main

import (
	"fmt"
	"os"

	"go-iot/internal/app"
	"go-iot/pkg/logger"
	"go-iot/pkg/option"
	// registry: blank-import plugin registration only (codec/servers/api init).
	// Startup order is owned by app.Start, not by init side effects beyond registration.
	_ "go-iot/pkg/registry"
)

func main() {
	opt := option.New()
	msg, err := opt.Parse()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", msg)
		os.Exit(1)
	}

	logger.Init(opt)
	defer logger.Sync()
	logger.Infof(opt.Banner, option.RELEASE, option.BUILD_TIME, option.COMMIT, option.REPO)

	application, err := app.New(opt)
	if err != nil {
		logger.Errorf("app new: %v", err)
		os.Exit(1)
	}

	// Run: Start (bootstrap + listen) then block on signal and Stop.
	if err := application.Run(); err != nil {
		logger.Errorf("app run: %v", err)
		os.Exit(1)
	}
}
