package main

import (
	"fmt"
	_ "go-iot/api"
	"go-iot/codec"
	"go-iot/models"
	_ "go-iot/network/clients/registry"
	_ "go-iot/network/servers/registry"
	_ "go-iot/notify/registry"
	"net/http"
	"runtime"
	"strings"

	"github.com/beego/beego/v2/core/config"
	"github.com/beego/beego/v2/core/logs"
	"github.com/beego/beego/v2/server/web"
	"github.com/beego/beego/v2/server/web/context"
)

func main() {

	configLog()
	setDefaultConfig()

	getConfigString("db.url", func(s string) {
		models.DefaultDbConfig.Url = s
	})
	models.InitDb()

	web.BConfig.RecoverFunc = defaultRecoverPanic
	web.ErrorHandler("404", func(rw http.ResponseWriter, r *http.Request) {
		rw.WriteHeader(404)
	})
	web.Run()
}

func defaultRecoverPanic(ctx *context.Context, cfg *web.Config) {
	if err := recover(); err != nil {
		if err == web.ErrAbort {
			return
		}
		logs.Error("the request url is ", ctx.Input.URL())
		var stack string
		for i := 1; ; i++ {
			_, file, line, ok := runtime.Caller(i)
			if !ok {
				break
			}
			logs.Error(fmt.Sprintf("%s:%d", file, line))
			stack = stack + fmt.Sprintf("%s:%d\n", file, line)
		}
		if ctx.Output.Status == 0 {
			ctx.Output.Status = 500
		}
		ctx.Output.JSON(models.JsonRespError(fmt.Errorf("%v", err)), false, false)
	}
}

func setDefaultConfig() {
	{
		getConfigString("device.manager.id", func(s string) {
			codec.DefaultManagerId = s
		})
		logs.Info("default device manager: ", codec.DefaultManagerId)
	}
	{
		getConfigString("es.url", func(s string) {
			codec.DefaultEsConfig.Url = s
		})
		getConfigString("es.usename", func(s string) {
			codec.DefaultEsConfig.Username = s
		})
		getConfigString("es.password", func(s string) {
			codec.DefaultEsConfig.Password = s
		})
		logs.Info("es config: ", codec.DefaultEsConfig)
	}
	{
		getConfigString("redis.addr", func(s string) {
			codec.DefaultRedisConfig.Addr = s
		})
		getConfigString("redis.password", func(s string) {
			codec.DefaultRedisConfig.Password = s
		})
		codec.DefaultRedisConfig.DB = getConfigInt("redis.db")
		logs.Info("redis config: ", codec.DefaultRedisConfig)
	}
}

func configLog() {
	logs.Async()
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
	var filename string = "go-iot.log"
	getConfigString("logs.filename", func(s string) {
		filename = "go-iot.log"
	})
	logs.GetBeeLogger().SetLevel(level)
	err := logs.SetLogger(logs.AdapterFile, fmt.Sprintf(`{"filename":"%s","level":%d,"maxlines":0,
	"maxsize":0,"daily":true,"maxdays":10,"color":false}`, filename, level))
	if err != nil {
		panic(err)
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
func getConfigInt(key string) int {
	data, err := config.Int(key)
	if err != nil {
		panic(err)
	}
	return data
}
