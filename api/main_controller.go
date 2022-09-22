package api

import (
	"github.com/beego/beego/v2/server/web"
)

func init() {
	web.Router("/", &MainController{})
}

type MainController struct {
	web.Controller
}

func (c *MainController) Get() {
	c.Data["Website"] = ""
	c.Data["Email"] = "gdouyang@foxmail.com"
	c.TplName = "index.html"
}