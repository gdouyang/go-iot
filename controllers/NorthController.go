package controllers

import (
	"encoding/json"
	"go-iot/agent"
	"go-iot/controllers/sender"
	"go-iot/models"
	"go-iot/models/operates"

	"github.com/astaxie/beego"
)

func init() {
	ns := beego.NewNamespace("/north/control",
		beego.NSRouter("/:id/switch", &NorthController{}, "post:Open"),
		beego.NSRouter("/:id/light", &NorthController{}, "post:Light"),
		beego.NSRouter("/status", &NorthController{}, "post:Status"))
	beego.AddNamespace(ns)
}

type NorthController struct {
	beego.Controller
}

// 设备开关
func (this *NorthController) Open() {
	deviceId := this.Ctx.Input.Param(":id")
	byteReq := this.Ctx.Input.RequestBody
	sender := sender.NorthSender{CheckAgent: true}
	sender.AgentFunc = func(device operates.Device) models.JsonResp {
		req := agent.NewRequest(device.Id, device.Sn, device.Provider, operates.OPER_OPEN, byteReq)
		res := agent.SendCommand(device.Agent, req)
		return res
	}
	this.Data["json"] = sender.Open(byteReq, deviceId)
	this.ServeJSON()
}

// 设备调光
func (this *NorthController) Light() {
	deviceId := this.Ctx.Input.Param(":id")
	byteReq := this.Ctx.Input.RequestBody

	sender := sender.NorthSender{CheckAgent: true}
	sender.AgentFunc = func(device operates.Device) models.JsonResp {
		req := agent.NewRequest(device.Id, device.Sn, device.Provider, operates.OPER_LIGHT, byteReq)
		res := agent.SendCommand(device.Agent, req)
		return res
	}
	this.Data["json"] = sender.Light(byteReq, deviceId)
	this.ServeJSON()
}

// 状态查询
func (this *NorthController) Status() {
	var ob map[string]interface{}
	json.Unmarshal(this.Ctx.Input.RequestBody, &ob)

	this.Data["json"] = &ob
	this.ServeJSON()
}
