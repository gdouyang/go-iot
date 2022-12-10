package main

import (
	"fmt"
	_ "go-iot/api"
	"go-iot/models"
	_ "go-iot/network/servers/registry"
	_ "go-iot/notify/registry"
	"net/http"

	"github.com/beego/beego/v2/core/config"
	"github.com/beego/beego/v2/core/logs"
	"github.com/beego/beego/v2/server/web"
)

func main() {
	{
		configLog()
	}
	dataSourceName, err := config.String("db.url")
	if err != nil {
		logs.Error("get dataSourceName failed")
	}
	models.DefaultDbConfig.Url = dataSourceName
	models.InitDb()
	web.ErrorHandler("404", func(rw http.ResponseWriter, r *http.Request) {
		rw.WriteHeader(404)
	})
	web.Run()
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
