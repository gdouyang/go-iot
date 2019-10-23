package controllers

import (
	"encoding/json"
	"go-iot/agent"
	"go-iot/models"
	"go-iot/models/modelfactory"
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
	beego.Info("deviceId=", deviceId)
	var ob []models.SwitchStatus
	byteReq := this.Ctx.Input.RequestBody
	json.Unmarshal(byteReq, &ob)

	device, err := modelfactory.GetDevice(deviceId)
	if err != nil {
		this.Data["json"] = models.JsonResp{Success: false, Msg: err.Error()}
	} else {
		p, err := operates.GetProvider(device.Provider)
		if err != nil {
			this.Data["json"] = models.JsonResp{Success: false, Msg: err.Error()}
		} else {
			if len(device.Agent) > 0 {
				req := agent.NewRequest(device.Id, device.Sn, device.Provider, operates.OPER_OPEN, byteReq)
				res := agent.SendCommand(device.Agent, req)
				this.Data["json"] = res
			} else {
				var switchOper operates.ISwitchOper
				switchOper = p.(operates.ISwitchOper)
				operResp := switchOper.Switch(ob, device)
				this.Data["json"] = models.JsonResp{Success: operResp.Success, Msg: operResp.Msg}
			}
		}
	}

	this.ServeJSON()
}

// 设备调光
func (this *NorthController) Light() {
	deviceId := this.Ctx.Input.Param(":id")
	beego.Info("deviceId=", deviceId)
	var ob map[string]int
	byteReq := this.Ctx.Input.RequestBody
	json.Unmarshal(byteReq, &ob)

	value := ob["value"]
	device, err := modelfactory.GetDevice(deviceId)
	if err != nil {
		this.Data["json"] = models.JsonResp{Success: false, Msg: err.Error()}
	} else {
		p, err := operates.GetProvider(device.Provider)
		if err != nil {
			this.Data["json"] = models.JsonResp{Success: false, Msg: err.Error()}
		} else {
			if len(device.Agent) > 0 {
				req := agent.NewRequest(device.Id, device.Sn, device.Provider, operates.OPER_LIGHT, byteReq)
				res := agent.SendCommand(device.Agent, req)
				this.Data["json"] = res
			} else {
				var lightOper operates.ILightOper
				lightOper = p.(operates.ILightOper)
				operResp := lightOper.Light(value, device)
				this.Data["json"] = models.JsonResp{Success: operResp.Success, Msg: operResp.Msg}
			}
		}
	}

	this.ServeJSON()
}

// 状态查询
func (this *NorthController) Status() {
	var ob map[string]interface{}
	json.Unmarshal(this.Ctx.Input.RequestBody, &ob)

	this.Data["json"] = &ob
	this.ServeJSON()
}
