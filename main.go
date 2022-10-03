package main

import (
	_ "go-iot/api"
	"go-iot/models"

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
	web.Run()
}
