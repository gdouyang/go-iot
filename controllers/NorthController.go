package controllers

import (
	"encoding/json"
	"go-iot/models"

	"github.com/astaxie/beego"
)

func init() {
	ns := beego.NewNamespace("/north/control",
		beego.NSRouter("/:id/switch", &NorthController{}, "post:Open"),
		beego.NSRouter("/status", &NorthController{}, "post:Status"))
	beego.AddNamespace(ns)
}

type NorthController struct {
	beego.Controller
}

// 设备开关
func (this *NorthController) Open() {
	deviceId := this.Ctx.Input.Param(":id")
	beego.Info("deviceId=", deviceId)
	var ob []models.SwitchStatus
	json.Unmarshal(this.Ctx.Input.RequestBody, &ob)

	var switchOper models.ISwitchOper
	p := models.GetProvider("xixunled")
	switchOper = p.(models.ISwitchOper)

	device := models.GetDevice(deviceId)
	switchOper.Switch(ob, device)

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
