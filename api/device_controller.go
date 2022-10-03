package api

import (
	"encoding/json"
	"go-iot/codec"
	"go-iot/codec/msg"
	"go-iot/models"
	device "go-iot/models/device"

	"github.com/beego/beego/v2/server/web"
)

// 设备管理
func init() {
	ns := web.NewNamespace("/api/device",
		web.NSRouter("/list", &DeviceController{}, "post:List"),
		web.NSRouter("/", &DeviceController{}, "post:Add"),
		web.NSRouter("/", &DeviceController{}, "put:Update"),
		web.NSRouter("/:id", &DeviceController{}, "delete:Delete"),
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
		ctl.Data["json"] = models.JsonRespError(err)
	} else {
		ctl.Data["json"] = &res
	}
	ctl.ServeJSON()
}

// 添加设备
func (ctl *DeviceController) Add() {
	defer ctl.ServeJSON()
	var ob models.Device
	json.Unmarshal(ctl.Ctx.Input.RequestBody, &ob)
	err := device.AddDevice(&ob)
	if err != nil {
		ctl.Data["json"] = models.JsonRespError(err)
		return
	}
	ctl.Data["json"] = models.JsonRespOk()
}

// 更新设备信息
func (ctl *DeviceController) Update() {
	defer ctl.ServeJSON()
	var ob models.Device
	json.Unmarshal(ctl.Ctx.Input.RequestBody, &ob)
	err := device.UpdateDevice(&ob)
	if err != nil {
		ctl.Data["json"] = models.JsonRespError(err)
		return
	}
	ctl.Data["json"] = models.JsonRespOk()
}

// 删除设备
func (ctl *DeviceController) Delete() {
	defer ctl.ServeJSON()
	var ob *models.Device = &models.Device{
		Id: ctl.Ctx.Input.Param(":id"),
	}
	err := device.DeleteDevice(ob)
	if err != nil {
		ctl.Data["json"] = models.JsonRespError(err)
		return
	}
	ctl.Data["json"] = models.JsonRespOk()
}

// 命令下发
func (ctl *DeviceController) CmdInvoke() {
	defer ctl.ServeJSON()

	var ob msg.FuncInvoke
	json.Unmarshal(ctl.Ctx.Input.RequestBody, &ob)
	device, err := device.GetDevice(ob.DeviceId)
	if err != nil {
		ctl.Data["json"] = models.JsonRespError(err)
		return
	}
	err = codec.DoCmdInvoke(device.ProductId, ob)
	if err != nil {
		ctl.Data["json"] = models.JsonRespError(err)
		return
	}
	ctl.Data["json"] = models.JsonRespOk()
}
