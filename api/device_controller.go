package api

import (
	"errors"
	"go-iot/codec"
	"go-iot/codec/msg"
	"go-iot/models"
	device "go-iot/models/device"
	"go-iot/models/network"
	"go-iot/network/clients"

	"github.com/beego/beego/v2/server/web"
)

var deviceResource = Resource{
	Id:   "device-mgr",
	Name: "设备",
	Action: []ResourceAction{
		QueryAction,
		CretaeAction,
		SaveAction,
		DeleteAction,
	},
}

// 设备管理
func init() {
	ns := web.NewNamespace("/api/device",
		web.NSRouter("/page", &DeviceController{}, "post:List"),
		web.NSRouter("/", &DeviceController{}, "post:Add"),
		web.NSRouter("/", &DeviceController{}, "put:Update"),
		web.NSRouter("/:id", &DeviceController{}, "delete:Delete"),
		web.NSRouter("/:id", &DeviceController{}, "get:GetOne"),
		web.NSRouter("/:id/connect", &DeviceController{}, "put:Connect"),
		web.NSRouter("/:id/disconnect", &DeviceController{}, "put:Disconnect"),
		web.NSRouter("/:id/deploy", &DeviceController{}, "put:Deploy"),
		web.NSRouter("/cmd", &DeviceController{}, "post:CmdInvoke"),
		web.NSRouter("/query-property/:id", &DeviceController{}, "get:QueryProperty"),
	)
	web.AddNamespace(ns)

	regResource(deviceResource)
}

type DeviceController struct {
	AuthController
}

// 查询设备列表
func (ctl *DeviceController) List() {
	if ctl.isForbidden(deviceResource, QueryAction) {
		return
	}
	var resp = models.JsonRespOk()
	defer func() {
		ctl.Data["json"] = resp
		ctl.ServeJSON()
	}()
	var ob models.PageQuery
	err := ctl.BindJSON(&ob)
	if err != nil {
		resp = models.JsonRespError(err)
		return
	}

	res, err := device.ListDevice(&ob)
	if err != nil {
		resp = models.JsonRespError(err)
		return
	}
	resp.Data = res
}

// 查询单个设备
func (ctl *DeviceController) GetOne() {
	if ctl.isForbidden(deviceResource, QueryAction) {
		return
	}
	var resp = models.JsonRespOk()
	defer func() {
		ctl.Data["json"] = resp
		ctl.ServeJSON()
	}()
	ob, err := device.GetDevice(ctl.Ctx.Input.Param(":id"))
	if err != nil {
		resp = models.JsonRespError(err)
		return
	}
	resp.Data = ob
}

// 添加设备
func (ctl *DeviceController) Add() {
	if ctl.isForbidden(deviceResource, CretaeAction) {
		return
	}
	var resp = models.JsonRespOk()
	defer func() {
		ctl.Data["json"] = resp
		ctl.ServeJSON()
	}()
	var ob models.Device
	err := ctl.BindJSON(&ob)
	if err != nil {
		resp = models.JsonRespError(err)
		return
	}
	err = device.AddDevice(&ob)
	if err != nil {
		resp = models.JsonRespError(err)
		return
	}
}

// 更新设备信息
func (ctl *DeviceController) Update() {
	if ctl.isForbidden(deviceResource, SaveAction) {
		return
	}
	var resp = models.JsonRespOk()
	defer func() {
		ctl.Data["json"] = resp
		ctl.ServeJSON()
	}()
	var ob models.Device
	err := ctl.BindJSON(&ob)
	if err != nil {
		resp = models.JsonRespError(err)
		return
	}
	err = device.UpdateDevice(&ob)
	if err != nil {
		resp = models.JsonRespError(err)
		return
	}
	resp = models.JsonRespOk()
}

// 删除设备
func (ctl *DeviceController) Delete() {
	if ctl.isForbidden(deviceResource, DeleteAction) {
		return
	}
	var resp = models.JsonRespOk()
	defer func() {
		ctl.Data["json"] = resp
		ctl.ServeJSON()
	}()
	var ob *models.Device = &models.Device{
		Id: ctl.Ctx.Input.Param(":id"),
	}
	err := device.DeleteDevice(ob)
	if err != nil {
		resp = models.JsonRespError(err)
		return
	}
}

// client设备连接
func (ctl *DeviceController) Connect() {
	if ctl.isForbidden(deviceResource, SaveAction) {
		return
	}
	var resp = models.JsonRespOk()
	defer func() {
		ctl.Data["json"] = resp
		ctl.ServeJSON()
	}()
	var ob *models.Device = &models.Device{
		Id: ctl.Ctx.Input.Param(":id"),
	}
	dev, err := device.GetDevice(ob.Id)
	if err != nil {
		resp = models.JsonRespError(err)
		return
	}
	n, err := network.GetByProductId(dev.ProductId)
	if err != nil {
		resp = models.JsonRespError(err)
		return
	}
	// 进行连接
	err = clients.Connect(ob.Id, convertCodecNetwork(*n))
	if err != nil {
		resp = models.JsonRespError(err)
		return
	}
}

func (ctl *DeviceController) Disconnect() {
	if ctl.isForbidden(deviceResource, SaveAction) {
		return
	}
	var resp = models.JsonRespOk()
	defer func() {
		ctl.Data["json"] = resp
		ctl.ServeJSON()
	}()
	var ob *models.Device = &models.Device{
		Id: ctl.Ctx.Input.Param(":id"),
	}
	dev, err := device.GetDevice(ob.Id)
	if err != nil {
		resp = models.JsonRespError(err)
		return
	}
	if dev == nil {
		resp = models.JsonRespError(errors.New("device not exist"))
		return
	}
	session := codec.GetSessionManager().Get(ob.Id)
	if session != nil {
		err := session.Disconnect()
		if err != nil {
			resp = models.JsonRespError(err)
			return
		}
	} else {
		resp = models.JsonRespError(errors.New("device is offline"))
		return
	}
}

func (ctl *DeviceController) Deploy() {
	if ctl.isForbidden(deviceResource, SaveAction) {
		return
	}
	var resp = models.JsonRespOk()
	defer func() {
		ctl.Data["json"] = resp
		ctl.ServeJSON()
	}()
	var ob *models.Device = &models.Device{
		Id: ctl.Ctx.Input.Param(":id"),
	}
	dev, err := device.GetDevice(ob.Id)
	if err != nil {
		resp = models.JsonRespError(err)
		return
	}
	if dev == nil {
		resp = models.JsonRespError(errors.New("device not exist"))
		return
	}
}

// 命令下发
func (ctl *DeviceController) CmdInvoke() {
	if ctl.isForbidden(deviceResource, SaveAction) {
		return
	}
	var resp = models.JsonRespOk()
	defer func() {
		ctl.Data["json"] = resp
		ctl.ServeJSON()
	}()

	var ob msg.FuncInvoke
	err := ctl.BindJSON(&ob)
	if err != nil {
		resp = models.JsonRespError(err)
		return
	}
	device, err := device.GetDevice(ob.DeviceId)
	if err != nil {
		resp = models.JsonRespError(err)
		return
	}
	err = codec.DoCmdInvoke(device.ProductId, ob)
	if err != nil {
		resp = models.JsonRespError(err)
		return
	}
}

// 查询设备属性
func (ctl *DeviceController) QueryProperty() {
	if ctl.isForbidden(deviceResource, QueryAction) {
		return
	}
	var resp = models.JsonRespOk()
	defer func() {
		ctl.Data["json"] = resp
		ctl.ServeJSON()
	}()

	var ob *models.Device = &models.Device{
		Id: ctl.Ctx.Input.Param(":id"),
	}
	device, err := device.GetDevice(ob.Id)
	if err != nil {
		resp = models.JsonRespError(err)
		return
	}
	product := codec.GetProductManager().Get(device.ProductId)
	if product == nil {
		resp = models.JsonRespError(errors.New("not found product"))
		return
	}
	param := map[string]interface{}{}
	param["deviceId"] = ob.Id
	res, err := product.GetTimeSeries().QueryProperty(product, param)
	if err != nil {
		resp = models.JsonRespError(err)
		return
	}
	resp.Data = res
}
