package main

import (
	"fmt"
	_ "go-iot/pkg/api"
	"go-iot/pkg/api/web"
	"go-iot/pkg/cluster"
	"go-iot/pkg/core"
	"go-iot/pkg/core/store"
	_ "go-iot/pkg/core/timeseries"
	"go-iot/pkg/es"
	"go-iot/pkg/logger"
	"go-iot/pkg/models"
	"go-iot/pkg/option"
	"go-iot/pkg/redis"
	_ "go-iot/pkg/registry"
	"os"
)

func main() {
	opt := option.New()
	msg, err := opt.Parse()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", msg)
		os.Exit(1)
	}
	// log config
	logger.Init(opt)
	defer logger.Sync()
	logger.Infof("RELEASE: %s", option.RELEASE)
	logger.Infof("BUILD_TIME: %s", option.BUILD_TIME)
	logger.Infof("COMMIT: %s", option.COMMIT)
	logger.Infof("REPO: %s", option.REPO)
	// configs
	core.RegDeviceStore(store.NewRedisStore())
	cluster.Config(opt)
	es.Config(opt)
	redis.Config(opt)
	// init db
	models.InitDb()

	web.MustNewServer(opt.APIAddr)
}
