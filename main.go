package main

import (
	"fmt"
	_ "go-iot/pkg/api"
	"go-iot/pkg/cluster"
	"go-iot/pkg/core"
	"go-iot/pkg/core/common"
	"go-iot/pkg/core/store"
	_ "go-iot/pkg/core/timeseries"
	"go-iot/pkg/es"
	"go-iot/pkg/logger"
	"go-iot/pkg/models"
	"go-iot/pkg/redis"
	_ "go-iot/pkg/registry"
	"net/http"
	"runtime"
	"strings"

	"github.com/beego/beego/v2/core/config"
	"github.com/beego/beego/v2/core/logs"
	"github.com/beego/beego/v2/server/web"
	"github.com/beego/beego/v2/server/web/context"
)

func main() {
	// goiot log config
	logger.Init(getConfigString)
	defer logger.Sync()
	{ // beego log config
		var logLevel string
		getConfigString("logs.level", func(s string) {
			logLevel = s
		})
		level := 7
		switch logLevel {
		case "info":
			level = logs.LevelInfo
		case "warn":
			level = logs.LevelWarn
		case "error":
			level = logs.LevelError
		}
		logs.GetBeeLogger().SetLevel(level)
	}
	// configs
	core.RegDeviceStore(store.NewRedisStore())
	cluster.Config(getConfigString)
	es.Config(getConfigString)
	redis.Config(getConfigString)
	// init db
	models.InitDb()

	web.BConfig.RecoverFunc = defaultRecoverPanic
	web.ErrorHandler("404", func(rw http.ResponseWriter, r *http.Request) {
		rw.WriteHeader(404)
	})
	// web.AddAPPStartHook(func() error {
	// 	logger.Infof("go iot is runing ")
	// 	return nil
	// })
	web.Run()
}

func defaultRecoverPanic(ctx *context.Context, cfg *web.Config) {
	if err := recover(); err != nil {
		if err == web.ErrAbort {
			return
		}
		logger.Errorf("the request url is %s", ctx.Input.URL())
		var stack string
		for i := 1; ; i++ {
			_, file, line, ok := runtime.Caller(i)
			if !ok {
				break
			}
			logger.Errorf(fmt.Sprintf("%s:%d", file, line))
			stack = stack + fmt.Sprintf("%s:%d\n", file, line)
		}
		if ctx.Output.Status == 0 {
			ctx.Output.Status = 500
		}
		ctx.Output.JSON(common.JsonRespError(fmt.Errorf("%v", err)), false, false)
	}
}

func getConfigString(key string, callback func(string)) {
	data, err := config.String(key)
	if err != nil {
		panic(err)
	}
	val := strings.TrimSpace(data)
	if len(val) > 0 {
		callback(val)
	}
}
