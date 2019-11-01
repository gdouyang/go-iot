package controllers

import (
	"encoding/json"
	"go-iot/controllers/sender"

	"github.com/astaxie/beego"
)

func init() {
	ns := beego.NewNamespace("/north/control",
		beego.NSRouter("/:id/switch", &NorthController{}, "post:Open"),
		beego.NSRouter("/:id/light", &NorthController{}, "post:Light"),
		beego.NSRouter("/status", &NorthController{}, "post:Status"))
	beego.AddNamespace(ns)
}

var (
	northSender sender.NorthSender = sender.NorthSender{CheckAgent: true}
)

type NorthController struct {
	beego.Controller
}

// 设备开关
func (this *NorthController) Open() {
	deviceId := this.Ctx.Input.Param(":id")
	byteReq := this.Ctx.Input.RequestBody
	this.Data["json"] = northSender.Open(byteReq, deviceId)
	this.ServeJSON()
}

// 设备调光
func (this *NorthController) Light() {
	deviceId := this.Ctx.Input.Param(":id")
	byteReq := this.Ctx.Input.RequestBody

	this.Data["json"] = northSender.Light(byteReq, deviceId)
	this.ServeJSON()
}

// 状态查询
func (this *NorthController) Status() {
	var ob map[string]interface{}
	json.Unmarshal(this.Ctx.Input.RequestBody, &ob)

	this.Data["json"] = &ob
	this.ServeJSON()
}
