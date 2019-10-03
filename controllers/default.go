package controllers

import (
	"github.com/astaxie/beego"

	"gopkg.in/mgo.v2"
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

//
func mongoExecute(cName string, exec func(collection *mgo.Collection)) {
	session, err := mgo.Dial("127.0.0.1") //Mongodb's connection
	if err != nil {
		panic(err)
	}
	session.SetMode(mgo.Monotonic, true)

	defer session.Close()
	exec(session.DB("iot").C(cName))
}
