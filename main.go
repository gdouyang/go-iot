package main

import (
	"fmt"
	"go-iot/pkg/api/web"
	"go-iot/pkg/cluster"
	"go-iot/pkg/core"
	"go-iot/pkg/es"
	"go-iot/pkg/logger"
	"go-iot/pkg/models"
	"go-iot/pkg/network/servers/goiot_mqtt5"
	"go-iot/pkg/option"
	"go-iot/pkg/redis"
	_ "go-iot/pkg/registry"
	"go-iot/pkg/ruleengine"
	"go-iot/pkg/store"
	"os"
)

func main() {
	opt := option.New()
	msg, err := opt.Parse()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", msg)
		os.Exit(1)
	}
	// 日志初始化
	logger.Init(opt)
	defer logger.Sync()
	logger.Infof(opt.Banner, option.RELEASE, option.BUILD_TIME, option.COMMIT, option.REPO)
	// 配置设备存储策略
	core.RegDeviceStore(store.NewRedisStore())
	// 集群配置
	cluster.Config(opt)
	// es配置
	es.Config(opt)
	// redis配置
	redis.Config(opt)
	// 规则引擎配置
	ruleengine.Config(opt)
	// 初始化数据库
	models.InitDb()
	// 启动mqtt5服务器
	err = goiot_mqtt5.Start()
	if err != nil {
		logger.Errorf("mqtt5 server start error: %v", err)
		panic(err)
	}
	// 启动web服务
	web.MustNewServer(opt.APIAddr)
}
