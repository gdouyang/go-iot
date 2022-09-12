package north

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
	c.Data["Website"] = "web.me"
	c.Data["Email"] = "astaxie@gmail.com"
	c.TplName = "index.html"
}
