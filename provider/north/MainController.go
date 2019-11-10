package north

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
	c.Data["Website"] = "beego.me"
	c.Data["Email"] = "astaxie@gmail.com"
	c.TplName = "index.html"
}
