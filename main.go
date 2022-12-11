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
	{
		configLog()
	}
	setEsConfig()
	dataSourceName, err := config.String("db.url")
	if err != nil {
		logs.Error("get dataSourceName failed")
	}

	models.DefaultDbConfig.Url = dataSourceName
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

func setEsConfig() {
	{
		esurl, err := config.String("es.url")
		if err != nil {
			logs.Error("get es.url failed")
		}
		codec.ES_URL = strings.TrimSpace(esurl)
	}
	{
		esusername, err := config.String("es.usename")
		if err != nil {
			logs.Error("get es.usename failed")
		}
		codec.ES_USERNAME = strings.TrimSpace(esusername)
	}
	{
		password, err := config.String("es.password")
		if err != nil {
			logs.Error("get es.password failed")
		}
		codec.ES_PASSWORD = strings.TrimSpace(password)
	}
}

func configLog() {
	logLevel, err := config.String("logs.level")
	if err != nil {
		logs.Error("get logs.level failed")
	}
	level := 7
	switch logLevel {
	case "info":
		level = logs.LevelInfo
	case "wran":
		level = logs.LevelWarn
	case "error":
		level = logs.LevelError
	}
	filename, err := config.String("logs.filename")
	if err != nil {
		logs.Error("get logs.filename failed")
	}
	if len(filename) == 0 {
		filename = "go-iot.log"
	}
	logs.GetBeeLogger().SetLevel(level)
	err = logs.SetLogger(logs.AdapterFile, fmt.Sprintf(`{"filename":"%s","level":%d,"maxlines":0,
	"maxsize":0,"daily":true,"maxdays":10,"color":false}`, filename, level))
	if err != nil {
		panic(err)
	}
}
