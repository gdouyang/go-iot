package api

import (
	"encoding/json"
	"errors"
	"go-iot/codec"
	"go-iot/codec/msg"
	"go-iot/models"
	device "go-iot/models/device"
	"go-iot/models/network"
	"go-iot/network/clients"

	"github.com/beego/beego/v2/server/web"
)

// 设备管理
func init() {
	ns := web.NewNamespace("/api/device",
		web.NSRouter("/list", &DeviceController{}, "post:List"),
		web.NSRouter("/", &DeviceController{}, "post:Add"),
		web.NSRouter("/", &DeviceController{}, "put:Update"),
		web.NSRouter("/:id", &DeviceController{}, "get:GetOne"),
		web.NSRouter("/:id", &DeviceController{}, "delete:Delete"),
		web.NSRouter("/connect/:id", &DeviceController{}, "put:Connect"),
		web.NSRouter("/cmd", &DeviceController{}, "post:CmdInvoke"),
		web.NSRouter("/query-property/:id", &DeviceController{}, "get:QueryProperty"),
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

// 查询单个设备
func (ctl *DeviceController) GetOne() {
	defer ctl.ServeJSON()
	ob, err := device.GetDevice(ctl.Ctx.Input.Param(":id"))
	if err != nil {
		ctl.Data["json"] = models.JsonRespError(err)
		return
	}

	ctl.Data["json"] = models.JsonRespOkData(ob)
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

// client设备连接
func (ctl *DeviceController) Connect() {
	defer ctl.ServeJSON()
	var ob *models.Device = &models.Device{
		Id: ctl.Ctx.Input.Param(":id"),
	}
	dev, err := device.GetDevice(ob.Id)
	if err != nil {
		ctl.Data["json"] = models.JsonRespError(err)
		return
	}
	n, err := network.GetByProductId(dev.ProductId)
	if err != nil {
		ctl.Data["json"] = models.JsonRespError(err)
		return
	}
	// 进行连接
	err = clients.Connect(ob.Id, convertCodecNetwork(n))
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

// 查询设备属性
func (ctl *DeviceController) QueryProperty() {
	defer ctl.ServeJSON()

	var ob *models.Device = &models.Device{
		Id: ctl.Ctx.Input.Param(":id"),
	}
	device, err := device.GetDevice(ob.Id)
	if err != nil {
		ctl.Data["json"] = models.JsonRespError(err)
		return
	}
	product := codec.GetProductManager().Get(device.ProductId)
	if product == nil {
		ctl.Data["json"] = models.JsonRespError(errors.New("not found product"))
		return
	}
	param := map[string]interface{}{}
	param["deviceId"] = ob.Id
	res, err := product.GetTimeSeries().QueryProperty(product, param)
	if err != nil {
		ctl.Data["json"] = models.JsonRespError(err)
		return
	}
	j := models.JsonRespOk()
	j.Data = res
	ctl.Data["json"] = j
}
