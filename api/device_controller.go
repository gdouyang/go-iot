package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"go-iot/codec"
	"go-iot/codec/msg"
	"go-iot/models"
	device "go-iot/models/device"
	"go-iot/models/network"
	"go-iot/network/clients"
	mqttclient "go-iot/network/clients/mqtt"
	tcpclient "go-iot/network/clients/tcp"
	"strconv"

	"github.com/beego/beego/v2/core/logs"
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
		web.NSRouter("/:id/detail", &DeviceController{}, "get:GetDetail"),
		web.NSRouter("/:id/connect", &DeviceController{}, "post:Connect"),
		web.NSRouter("/:id/disconnect", &DeviceController{}, "post:Disconnect"),
		web.NSRouter("/:id/deploy", &DeviceController{}, "post:Deploy"),
		web.NSRouter("/:id/undeploy", &DeviceController{}, "post:Undeploy"),
		web.NSRouter("/:id/cmd", &DeviceController{}, "post:CmdInvoke"),
		web.NSRouter("/propertys/:id/query", &DeviceController{}, "post:QueryProperty"),
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
	var ob models.PageQuery
	err := ctl.BindJSON(&ob)
	if err != nil {
		ctl.RespError(err)
		return
	}

	res, err := device.ListDevice(&ob)
	if err != nil {
		ctl.RespError(err)
		return
	}
	ctl.RespOkData(res)
}

// 查询单个设备
func (ctl *DeviceController) GetOne() {
	if ctl.isForbidden(deviceResource, QueryAction) {
		return
	}
	ob, err := device.GetDeviceMust(ctl.Ctx.Input.Param(":id"))
	if err != nil {
		ctl.RespError(err)
		return
	}
	ctl.RespOkData(ob)
}

func (ctl *DeviceController) GetDetail() {
	if ctl.isForbidden(deviceResource, QueryAction) {
		return
	}
	ob, err := device.GetDeviceMust(ctl.Ctx.Input.Param(":id"))
	if err != nil {
		ctl.RespError(err)
		return
	}
	product, err := device.GetProductMust(ob.ProductId)
	if err != nil {
		ctl.RespError(err)
		return
	}
	nw, err := network.GetByProductId(ob.ProductId)
	if err != nil {
		ctl.RespError(err)
		return
	}
	var alins = struct {
		models.DeviceModel
		Metadata    string `json:"metadata"`
		ProductName string `json:"productName"`
		NetworkType string `json:"networkType"`
	}{}
	alins.Metadata = product.Metadata
	alins.ProductName = product.Name
	if nw != nil {
		alins.NetworkType = nw.Type
	}
	alins.DeviceModel = *ob
	if ob.State != models.NoActive {
		alins.State = models.OFFLINE
		sess := codec.GetSessionManager().Get(ob.Id)
		if sess != nil {
			alins.State = models.ONLINE
		} else {
			liefcycle := codec.GetDeviceLifeCycle(ob.ProductId)
			if liefcycle != nil {
				state, err := liefcycle.OnStateChecker(&codec.BaseContext{
					ProductId: ob.ProductId,
					DeviceId:  ob.Id,
				})
				if err == nil {
					if state == models.ONLINE {
						alins.State = models.ONLINE
					}
				} else {
					logs.Warn("OnStateChecker device [%s] err: ", ob.Id, err)
				}
			}
		}
		device.UpdateOnlineStatus(ob.Id, alins.State)
	}
	ctl.RespOkData(alins)
}

// 添加设备
func (ctl *DeviceController) Add() {
	if ctl.isForbidden(deviceResource, CretaeAction) {
		return
	}
	var ob models.Device
	err := ctl.BindJSON(&ob)
	if err != nil {
		ctl.RespError(err)
		return
	}
	err = device.AddDevice(&ob)
	if err != nil {
		ctl.RespError(err)
		return
	}
	ctl.RespOk()
}

// 更新设备信息
func (ctl *DeviceController) Update() {
	if ctl.isForbidden(deviceResource, SaveAction) {
		return
	}
	var ob models.DeviceModel
	err := ctl.BindJSON(&ob)
	if err != nil {
		ctl.RespError(err)
		return
	}
	en := ob.ToEnitty()
	err = device.UpdateDevice(&en)
	if err != nil {
		ctl.RespError(err)
		return
	}
	ctl.RespOk()
}

// 删除设备
func (ctl *DeviceController) Delete() {
	if ctl.isForbidden(deviceResource, DeleteAction) {
		return
	}
	var ob *models.Device = &models.Device{
		Id: ctl.Ctx.Input.Param(":id"),
	}
	err := device.DeleteDevice(ob)
	if err != nil {
		ctl.RespError(err)
		return
	}
	ctl.RespOk()
}

// client设备连接
func (ctl *DeviceController) Connect() {
	if ctl.isForbidden(deviceResource, SaveAction) {
		return
	}
	var ob *models.Device = &models.Device{
		Id: ctl.Ctx.Input.Param(":id"),
	}
	err := connectClientDevice(ob.Id)
	if err != nil {
		ctl.RespError(err)
		return
	}
	ctl.RespOk()
}

func connectClientDevice(deviceId string) error {
	dev, err := device.GetDeviceMust(deviceId)
	if err != nil {
		return err
	}
	nw, err := network.GetByProductId(dev.ProductId)
	if err != nil {
		return err
	}
	if nw == nil {
		return fmt.Errorf("product [%s] not have network config", dev.ProductId)
	}
	if !codec.IsNetClientType(nw.Type) {
		return errors.New("only client type net can do it")
	}
	// 进行连接
	devoper := codec.GetDeviceManager().Get(deviceId)
	if devoper == nil {
		return errors.New("devoper is nil")
	}
	if codec.TCP_CLIENT == codec.NetClientType(nw.Type) {
		spec := &tcpclient.TcpClientSpec{}
		err = spec.FromJson(nw.Configuration)
		if err != nil {
			return err
		}
		spec.Host = devoper.GetConfig("host")
		port, err := strconv.Atoi(devoper.GetConfig("port"))
		if err != nil {
			return errors.New("port is not number")
		}
		spec.Port = int32(port)
		b, _ := json.Marshal(spec)
		nw.Configuration = string(b)
	} else if codec.MQTT_CLIENT == codec.NetClientType(nw.Type) {
		spec := &mqttclient.MQTTClientSpec{}
		err = spec.FromJson(nw.Configuration)
		if err != nil {
			return err
		}
		spec.Host = devoper.GetConfig("host")
		port, err := strconv.Atoi(devoper.GetConfig("port"))
		if err != nil {
			return errors.New("port is not number")
		}
		spec.Port = int32(port)
		spec.ClientId = devoper.GetConfig("clientId")
		spec.Username = devoper.GetConfig("username")
		spec.Password = devoper.GetConfig("password")
		b, _ := json.Marshal(spec)
		nw.Configuration = string(b)
	}
	err = clients.Connect(deviceId, convertCodecNetwork(*nw))
	if err != nil {
		return err
	}
	err = device.UpdateOnlineStatus(deviceId, models.ONLINE)
	if err != nil {
		return err
	}
	return nil
}

func (ctl *DeviceController) Disconnect() {
	if ctl.isForbidden(deviceResource, SaveAction) {
		return
	}
	var ob *models.Device = &models.Device{
		Id: ctl.Ctx.Input.Param(":id"),
	}
	_, err := device.GetDeviceMust(ob.Id)
	if err != nil {
		ctl.RespError(err)
		return
	}
	session := codec.GetSessionManager().Get(ob.Id)
	if session != nil {
		err := session.Disconnect()
		if err != nil {
			ctl.RespError(err)
			return
		}
	} else {
		ctl.RespError(errors.New("device is offline"))
		return
	}
	ctl.RespOk()
}

func (ctl *DeviceController) Deploy() {
	if ctl.isForbidden(deviceResource, SaveAction) {
		return
	}
	var ob *models.Device = &models.Device{
		Id: ctl.Ctx.Input.Param(":id"),
	}
	dev, err := device.GetDeviceMust(ob.Id)
	if err != nil {
		ctl.RespError(err)
		return
	}
	if len(dev.State) == 0 || dev.State == models.NoActive {
		device.UpdateOnlineStatus(ob.Id, models.OFFLINE)
	}
	// TODO
	ctl.RespOk()
}

func (ctl *DeviceController) Undeploy() {
	if ctl.isForbidden(deviceResource, SaveAction) {
		return
	}
	var ob *models.Device = &models.Device{
		Id: ctl.Ctx.Input.Param(":id"),
	}
	_, err := device.GetDeviceMust(ob.Id)
	if err != nil {
		ctl.RespError(err)
		return
	}
	device.UpdateOnlineStatus(ob.Id, models.NoActive)
	// TODO
	ctl.RespOk()
}

// 命令下发
func (ctl *DeviceController) CmdInvoke() {
	if ctl.isForbidden(deviceResource, SaveAction) {
		return
	}
	deviceId := ctl.Ctx.Input.Param(":id")

	var ob msg.FuncInvoke
	err := ctl.BindJSON(&ob)
	if err != nil {
		ctl.RespError(err)
		return
	}
	ob.DeviceId = deviceId
	device, err := device.GetDeviceMust(ob.DeviceId)
	if err != nil {
		ctl.RespError(err)
		return
	}
	err = codec.DoCmdInvoke(device.ProductId, ob)
	if err != nil {
		ctl.RespError(err)
		return
	}
	ctl.RespOk()
}

// 查询设备属性
func (ctl *DeviceController) QueryProperty() {
	if ctl.isForbidden(deviceResource, QueryAction) {
		return
	}

	var ob *models.Device = &models.Device{
		Id: ctl.Ctx.Input.Param(":id"),
	}
	var param codec.QueryParam
	err := ctl.BindJSON(&param)
	if err != nil {
		ctl.RespError(err)
		return
	}
	param.DeviceId = ob.Id

	device, err := device.GetDeviceMust(ob.Id)
	if err != nil {
		ctl.RespError(err)
		return
	}
	product := codec.GetProductManager().Get(device.ProductId)
	if product == nil {
		ctl.RespError(fmt.Errorf("not found product %s", device.ProductId))
		return
	}
	res, err := product.GetTimeSeries().QueryProperty(product, param)
	if err != nil {
		ctl.RespError(err)
		return
	}
	ctl.RespOkData(res)
}
