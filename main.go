package main

import (
	"fmt"
	_ "go-iot/api"
	"go-iot/codec"
	"go-iot/models"
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

	models.DefaultDbConfig.Url = getConfigString("db.url")
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
		codec.DefaultManagerId = getConfigString("device.manager.id")
	}
	{
		codec.DefaultEsConfig.Url = getConfigString("es.url")
		codec.DefaultEsConfig.Username = getConfigString("es.usename")
		codec.DefaultEsConfig.Password = getConfigString("es.password")
	}
	{
		codec.DefaultRedisConfig.Addr = getConfigString("redis.addr")
		codec.DefaultRedisConfig.Password = getConfigString("redis.password")
		codec.DefaultRedisConfig.DB = getConfigInt("redis.db")
	}
}

func configLog() {
	logLevel := getConfigString("logs.level")
	level := 7
	switch logLevel {
	case "info":
		level = logs.LevelInfo
	case "warn":
		level = logs.LevelWarn
	case "error":
		level = logs.LevelError
	}
	filename := getConfigString("logs.filename")
	if len(filename) == 0 {
		filename = "go-iot.log"
	}
	logs.GetBeeLogger().SetLevel(level)
	err := logs.SetLogger(logs.AdapterFile, fmt.Sprintf(`{"filename":"%s","level":%d,"maxlines":0,
	"maxsize":0,"daily":true,"maxdays":10,"color":false}`, filename, level))
	if err != nil {
		panic(err)
	}
}

func getConfigString(key string) string {
	data, err := config.String(key)
	if err != nil {
		panic(err)
	}
	return strings.TrimSpace(data)
}
func getConfigInt(key string) int {
	data, err := config.Int(key)
	if err != nil {
		panic(err)
	}
	return data
}
