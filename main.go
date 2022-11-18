package main

import (
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
