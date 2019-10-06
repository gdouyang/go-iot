package controllers

import (
	"github.com/astaxie/beego"
)

func init() {
	beego.Router("/", &MainController{})
}

type MainController struct {
	beego.Controller
}

func (c *MainController) Get() {
	// beego.Info(beego.AppConfig.String("appname"))
	c.Data["Website"] = "beego.me"
	c.Data["Email"] = "astaxie@gmail.com"
	c.TplName = "index.html"
}
