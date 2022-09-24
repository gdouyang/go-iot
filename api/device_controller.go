package api

import (
	"encoding/json"
	"go-iot/models"
	device "go-iot/models/device"
	"go-iot/provider/codec"
	"go-iot/provider/codec/msg"

	"github.com/beego/beego/v2/server/web"
)

// 设备管理
func init() {
	ns := web.NewNamespace("/api/device",
		web.NSRouter("/list", &DeviceController{}, "post:List"),
		web.NSRouter("/", &DeviceController{}, "post:Add"),
		web.NSRouter("/?:id", &DeviceController{}, "delete:Delete"),
		web.NSRouter("/", &DeviceController{}, "put:Update"),
		web.NSRouter("/cmd", &DeviceController{}, "post:CmdInvoke"),
	)
	web.AddNamespace(ns)
}

type DeviceController struct {
	web.Controller
}

// 查询设备列表
func (ctl *DeviceController) List() {
	var ob models.PageQuery
	json.Unmarshal(ctl.Ctx.Input.RequestBody, &ob)

	res, err := device.ListDevice(&ob)
	if err != nil {
		ctl.Data["json"] = models.JsonResp{Success: false, Msg: err.Error()}
	} else {

		ctl.Data["json"] = &res
	}
	ctl.ServeJSON()
}

// 添加设备
func (ctl *DeviceController) Add() {
	var ob models.Device
	json.Unmarshal(ctl.Ctx.Input.RequestBody, &ob)
	ctl.Data["json"] = device.AddDevice(&ob)
	ctl.ServeJSON()
}

// 更新设备信息
func (ctl *DeviceController) Update() {
	var ob models.Device
	json.Unmarshal(ctl.Ctx.Input.RequestBody, &ob)
	ctl.Data["json"] = device.UpdateDevice(&ob)
	ctl.ServeJSON()
}

// 删除设备
func (ctl *DeviceController) Delete() {
	var ob *models.Device = &models.Device{
		Id: ctl.Ctx.Input.Param(":id"),
	}
	ctl.Data["json"] = device.DeleteDevice(ob)
	ctl.ServeJSON()
}

// 命令下发
func (ctl *DeviceController) CmdInvoke() {
	var ob msg.FuncInvoke
	json.Unmarshal(ctl.Ctx.Input.RequestBody, &ob)
	device, err := device.GetDevice(ob.DeviceId)
	if err != nil {
		ctl.Data["json"] = err
	}
	codec.DoFuncInvoke(device.ProductId, ob)
	ctl.Data["json"] = ""
	ctl.ServeJSON()
}
