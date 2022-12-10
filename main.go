package main

import (
	"fmt"
	_ "go-iot/api"
	"go-iot/codec"
	"go-iot/models"
	_ "go-iot/network/servers/registry"
	_ "go-iot/notify/registry"
	"net/http"
	"strings"

	"github.com/beego/beego/v2/core/config"
	"github.com/beego/beego/v2/core/logs"
	"github.com/beego/beego/v2/server/web"
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
	web.ErrorHandler("404", func(rw http.ResponseWriter, r *http.Request) {
		rw.WriteHeader(404)
	})
	web.Run()
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
