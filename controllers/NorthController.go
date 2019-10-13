package controllers

import (
	"encoding/json"
	"go-iot/models"
	"go-iot/models/operates"

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

	var switchOper operates.ISwitchOper
	p := operates.GetProvider("xixunled")
	switchOper = p.(operates.ISwitchOper)

	device, err := models.GetDevice(deviceId)
	if err != nil {
		this.Data["json"] = models.JsonResp{Success: false, Msg: err.Error()}
	} else {
		operResp := switchOper.Switch(ob, device)
		this.Data["json"] = models.JsonResp{Success: operResp.Success, Msg: operResp.Msg}
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
