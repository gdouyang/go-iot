package controllers

import (
	"encoding/json"
	"go-iot/agent"
	"go-iot/controllers/sender"
	"go-iot/models"
	"go-iot/models/led"
	"go-iot/models/operates"

	"github.com/astaxie/beego"
)

// 设备管理
func init() {
	beego.Router("/led/list", &LedController{}, "post:List")
	beego.Router("/led/add", &LedController{}, "post:Add")
	beego.Router("/led/update", &LedController{}, "post:Update")
	beego.Router("/led/delete", &LedController{}, "post:Delete")
	beego.Router("/led/listProvider", &LedController{}, "post:ListProvider")
}

type LedController struct {
	beego.Controller
}

// 查询设备列表
func (this *LedController) List() {
	var ob models.PageQuery
	json.Unmarshal(this.Ctx.Input.RequestBody, &ob)

	res, err := led.ListDevice(&ob)
	if err != nil {
		this.Data["json"] = models.JsonResp{Success: false, Msg: err.Error()}
	} else {

		this.Data["json"] = &res
	}
	this.ServeJSON()
}

// 添加设备
func (this *LedController) Add() {
	data := this.Ctx.Input.RequestBody

	ledSender := sender.LedSender{CheckAgent: true}
	ledSender.AgentFunc = func(device led.Device) models.JsonResp {
		req := agent.NewRequest(device.Id, device.Sn, device.Provider, sender.LED_ADD, data)
		res := agent.SendCommand(device.Agent, req)
		return res
	}

	this.Data["json"] = ledSender.Add(data)
	this.ServeJSON()
}

// 更新设备信息
func (this *LedController) Update() {
	data := this.Ctx.Input.RequestBody

	ledSender := sender.LedSender{CheckAgent: true}
	ledSender.AgentFunc = func(device led.Device) models.JsonResp {
		req := agent.NewRequest(device.Id, device.Sn, device.Provider, sender.LED_UPDATE, data)
		res := agent.SendCommand(device.Agent, req)
		return res
	}
	this.Data["json"] = ledSender.Update(data)
	this.ServeJSON()
}

// 删除设备
func (this *LedController) Delete() {
	data := this.Ctx.Input.RequestBody
	ledSender := sender.LedSender{CheckAgent: true}
	ledSender.AgentFunc = func(device led.Device) models.JsonResp {
		req := agent.NewRequest(device.Id, device.Sn, device.Provider, sender.LED_DELETE, data)
		res := agent.SendCommand(device.Agent, req)
		return res
	}
	this.Data["json"] = ledSender.Delete(data)
	this.ServeJSON()
}

// 查询所有厂商
func (this *LedController) ListProvider() {
	pros := operates.AllProvierId()
	this.Data["json"] = &pros
	this.ServeJSON()
}
