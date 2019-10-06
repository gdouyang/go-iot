package controllers

import (
	"encoding/json"

	"github.com/astaxie/beego"
)

func init() {
	ns := beego.NewNamespace("/north/control",
		beego.NSRouter("/open", &NorthController{}, "post:Open"),
		beego.NSRouter("/status", &NorthController{}, "post:Status"))
	beego.AddNamespace(ns)
}

type NorthController struct {
	beego.Controller
}

// 设备开关
func (this *NorthController) Open() {
	var ob map[string]interface{}
	json.Unmarshal(this.Ctx.Input.RequestBody, &ob)

	this.Data["json"] = &ob
	this.ServeJSON()
}

// 状态查询
func (this *NorthController) Status() {
	var ob map[string]interface{}
	json.Unmarshal(this.Ctx.Input.RequestBody, &ob)

	this.Data["json"] = &ob
	this.ServeJSON()
}
