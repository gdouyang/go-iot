package api

import (
	"errors"
	"fmt"
	"go-iot/pkg/core"
	"go-iot/pkg/core/msg"
	"go-iot/pkg/models"
	device "go-iot/pkg/models/device"
	"go-iot/pkg/models/network"
	"go-iot/pkg/network/clients"
	"time"

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
		ImportAction,
	},
}

// 设备管理
func init() {
	ns := web.NewNamespace("/api/device",
		web.NSRouter("/page", &DeviceController{}, "post:Page"),
		web.NSRouter("/page-es", &DeviceController{}, "post:PageEs"),
		web.NSRouter("/", &DeviceController{}, "post:Add"),
		web.NSRouter("/", &DeviceController{}, "put:Update"),
		web.NSRouter("/:id", &DeviceController{}, "delete:Delete"),
		web.NSRouter("/:id", &DeviceController{}, "get:GetOne"),
		web.NSRouter("/:id/detail", &DeviceController{}, "get:GetDetail"),
		web.NSRouter("/:id/connect", &DeviceController{}, "post:Connect"),
		web.NSRouter("/:id/disconnect", &DeviceController{}, "post:Disconnect"),
		web.NSRouter("/:id/deploy", &DeviceController{}, "post:Deploy"),
		web.NSRouter("/:id/undeploy", &DeviceController{}, "post:Undeploy"),
		web.NSRouter("/batch/_deploy", &DeviceController{}, "post:BatchDeploy"),
		web.NSRouter("/batch/_undeploy", &DeviceController{}, "post:BatchUndeploy"),
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
func (ctl *DeviceController) Page() {
	if ctl.isForbidden(deviceResource, QueryAction) {
		return
	}
	var ob models.PageQuery[models.Device]
	err := ctl.BindJSON(&ob)
	if err != nil {
		ctl.RespError(err)
		return
	}

	res, err := device.PageDevice(&ob, ctl.GetCurrentUser().Id)
	if err != nil {
		ctl.RespError(err)
		return
	}
	ctl.RespOkData(res)
}

// 查询设备列表
func (ctl *DeviceController) PageEs() {
	if ctl.isForbidden(deviceResource, QueryAction) {
		return
	}
	var ob models.PageQuery[models.DeviceModel]
	err := ctl.BindJSON(&ob)
	if err != nil {
		ctl.RespError(err)
		return
	}

	res, err := device.PageDeviceEs(&ob, ctl.GetCurrentUser().Id)
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
	deviceId := ctl.Param(":id")
	ob, err := ctl.getDeviceAndCheckCreateId(deviceId)
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
	ob, err := ctl.getDeviceAndCheckCreateId(ctl.Param(":id"))
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
		sess := core.GetSession(ob.Id)
		if sess != nil {
			alins.State = models.ONLINE
		} else {
			liefcycle := core.GetDeviceLifeCycle(ob.ProductId)
			if liefcycle != nil {
				state, err := liefcycle.OnStateChecker(&core.BaseContext{
					ProductId: ob.ProductId,
					DeviceId:  ob.Id,
				})
				if err == nil {
					if state == models.ONLINE {
						alins.State = models.ONLINE
					}
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
	var ob models.DeviceModel
	err := ctl.BindJSON(&ob)
	if err != nil {
		ctl.RespError(err)
		return
	}
	ob.CreateId = ctl.GetCurrentUser().Id
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
	_, err = ctl.getDeviceAndCheckCreateId(ob.Id)
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
	deviceId := ctl.Param(":id")
	_, err := ctl.getDeviceAndCheckCreateId(deviceId)
	if err != nil {
		ctl.RespError(err)
		return
	}
	err = device.DeleteDevice(deviceId)
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
	deviceId := ctl.Param(":id")
	_, err := ctl.getDeviceAndCheckCreateId(deviceId)
	if err != nil {
		ctl.RespError(err)
		return
	}
	err = connectClientDevice(deviceId)
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
	// 进行连接
	devoper := core.GetDevice(deviceId)
	if devoper == nil {
		return errors.New("devoper is nil")
	}
	conf, err := convertCodecNetwork(*nw)
	if err != nil {
		return err
	}
	err = clients.Connect(deviceId, conf)
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
	deviceId := ctl.Param(":id")
	_, err := ctl.getDeviceAndCheckCreateId(deviceId)
	if err != nil {
		ctl.RespError(err)
		return
	}
	session := core.GetSession(deviceId)
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
	deviceId := ctl.Param(":id")
	ctl.enable(deviceId, true)
	ctl.RespOk()
}

func (ctl *DeviceController) Undeploy() {
	if ctl.isForbidden(deviceResource, SaveAction) {
		return
	}
	deviceId := ctl.Param(":id")
	ctl.enable(deviceId, false)
	ctl.RespOk()
}

func (ctl *DeviceController) BatchDeploy() {
	if ctl.isForbidden(deviceResource, SaveAction) {
		return
	}
	var deviceIds []string
	ctl.BindJSON(&deviceIds)
	token := fmt.Sprintf("%v", time.Now().UnixMicro())
	setSseData(token, "")
	go func() {
		total := 0
		resp := `{"success":true, "result": {"finish": %v, "num": %d}}`
		if len(deviceIds) > 0 {
			for _, deviceId := range deviceIds {
				ctl.enable(deviceId, true)
				total = total + 1
				if total%5 == 0 {
					setSseData(token, fmt.Sprintf(resp, false, total))
				}
			}
			setSseData(token, fmt.Sprintf(resp, true, total))
		} else {
			condition := models.Device{State: models.NoActive}
			for {
				var page *models.PageQuery[models.Device] = &models.PageQuery[models.Device]{PageSize: 300, PageNum: 1, Condition: condition}
				result, err := device.PageDevice(page, ctl.GetCurrentUser().Id)
				if err != nil {
					logs.Error(err)
					break
				}
				list := result.List
				if len(list) == 0 {
					break
				}
				var ids []string
				for _, dev := range list {
					ids = append(ids, dev.Id)
					devopr := core.GetDevice(dev.Id)
					if devopr == nil {
						devopr = core.NewDevice(dev.Id, dev.ProductId, dev.CreateId)
					}
					model := models.DeviceModel{}
					model.FromEnitty(dev)
					devopr.Config = model.Metaconfig
					core.PutDevice(devopr)
				}
				err = device.UpdateOnlineStatusList(ids, models.OFFLINE)
				if err != nil {
					logs.Error(err)
				} else {
					total = total + len(list)
				}
				setSseData(token, fmt.Sprintf(resp, false, total))
			}
			setSseData(token, fmt.Sprintf(resp, true, total))
			logs.Info("batch deploy done")
		}
	}()
	ctl.RespOkData(token)
}

func (ctl *DeviceController) BatchUndeploy() {
	if ctl.isForbidden(deviceResource, SaveAction) {
		return
	}
	var deviceIds []string
	ctl.BindJSON(&deviceIds)
	if len(deviceIds) == 0 {
		ctl.RespError(errors.New("ids must be persent"))
	}
	for _, deviceId := range deviceIds {
		ctl.enable(deviceId, false)
	}
	ctl.RespOk()
}

func (ctl *DeviceController) enable(deviceId string, isDeploy bool) {
	dev, err := ctl.getDeviceAndCheckCreateId(deviceId)
	if err != nil {
		ctl.RespError(err)
		return
	}
	if isDeploy {
		if len(dev.State) == 0 || dev.State == models.NoActive {
			device.UpdateOnlineStatus(deviceId, models.OFFLINE)
		}
		devopr := core.GetDevice(dev.Id)
		if devopr == nil {
			devopr = core.NewDevice(dev.Id, dev.ProductId, dev.CreateId)
		}
		devopr.Config = dev.Metaconfig
		core.PutDevice(devopr)
	} else {
		device.UpdateOnlineStatus(deviceId, models.NoActive)
		core.DeleteDevice(deviceId)
	}
}

// 命令下发
func (ctl *DeviceController) CmdInvoke() {
	if ctl.isForbidden(deviceResource, SaveAction) {
		return
	}
	deviceId := ctl.Param(":id")

	var ob msg.FuncInvoke
	err := ctl.BindJSON(&ob)
	if err != nil {
		ctl.RespError(err)
		return
	}
	ob.DeviceId = deviceId
	device, err := ctl.getDeviceAndCheckCreateId(ob.DeviceId)
	if err != nil {
		ctl.RespError(err)
		return
	}
	err = core.DoCmdInvoke(device.ProductId, ob)
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

	deviceId := ctl.Param(":id")
	var param core.QueryParam
	err := ctl.BindJSON(&param)
	if err != nil {
		ctl.RespError(err)
		return
	}
	param.DeviceId = deviceId

	device, err := ctl.getDeviceAndCheckCreateId(deviceId)
	if err != nil {
		ctl.RespError(err)
		return
	}
	product := core.GetProduct(device.ProductId)
	if product == nil {
		ctl.RespError(fmt.Errorf("not found product %s, make sure product is deployed", device.ProductId))
		return
	}
	res, err := product.GetTimeSeries().QueryProperty(product, param)
	if err != nil {
		ctl.RespError(err)
		return
	}
	ctl.RespOkData(res)
}

func (ctl *DeviceController) getDeviceAndCheckCreateId(deviceId string) (*models.DeviceModel, error) {
	ob, err := device.GetDeviceMust(deviceId)
	if err != nil {
		return nil, err
	}
	if ob.CreateId != ctl.GetCurrentUser().Id {
		return nil, errors.New("device is not you created")
	}
	return ob, nil
}
