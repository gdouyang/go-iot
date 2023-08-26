package api

import (
	"errors"
	"fmt"
	"go-iot/pkg/api/web"
	"go-iot/pkg/cluster"
	"go-iot/pkg/core"
	"go-iot/pkg/core/common"
	"go-iot/pkg/models"
	device "go-iot/pkg/models/device"
	"go-iot/pkg/models/network"
	"time"

	logs "go-iot/pkg/logger"
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
	web.RegisterAPI("/device/page", "POST", &DeviceController{}, "Page")
	web.RegisterAPI("/device", "POST", &DeviceController{}, "Add")
	web.RegisterAPI("/device", "PUT", &DeviceController{}, "Update")
	web.RegisterAPI("/device/{id}", "DELETE", &DeviceController{}, "Delete")
	web.RegisterAPI("/device/{id}", "GET", &DeviceController{}, "GetOne")
	web.RegisterAPI("/device/{id}/detail", "GET", &DeviceController{}, "GetDetail")
	web.RegisterAPI("/device/{id}/connect", "POST", &DeviceController{}, "Connect")
	web.RegisterAPI("/device/{id}/disconnect", "POST", &DeviceController{}, "Disconnect")
	web.RegisterAPI("/device/{id}/deploy", "POST", &DeviceController{}, "Deploy")
	web.RegisterAPI("/device/{id}/undeploy", "POST", &DeviceController{}, "Undeploy")
	web.RegisterAPI("/device/batch/deploy", "POST", &DeviceController{}, "BatchDeploy")
	web.RegisterAPI("/device/batch/undeploy", "POST", &DeviceController{}, "BatchUndeploy")
	web.RegisterAPI("/device/{id}/invoke", "POST", &DeviceController{}, "CmdInvoke")
	web.RegisterAPI("/device/{id}/properties", "POST", &DeviceController{}, "QueryProperty")
	web.RegisterAPI("/device/{id}/logs", "POST", &DeviceController{}, "QueryLogs")
	web.RegisterAPI("/device/{id}/event/{eventId}", "POST", &DeviceController{}, "QueryEvent")

	RegResource(deviceResource)
}

type DeviceController struct {
	AuthController
}

// 查询设备列表
func (ctl *DeviceController) Page() {
	if ctl.isForbidden(deviceResource, QueryAction) {
		return
	}
	var ob models.PageQuery
	err := ctl.BindJSON(&ob)
	if err != nil {
		ctl.RespError(err)
		return
	}

	res, err := device.PageDevice(&ob, &ctl.GetCurrentUser().Id)
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
	deviceId := ctl.Param("id")
	ob, err := ctl.getDeviceAndCheckCreateId(deviceId)
	if err != nil {
		ctl.RespError(err)
		return
	}
	ctl.RespOkData(ob)
}

// get device detail info
func (ctl *DeviceController) GetDetail() {
	if ctl.isForbidden(deviceResource, QueryAction) {
		return
	}
	ob, err := ctl.getDeviceAndCheckCreateId(ctl.Param("id"))
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
	if ob.State != core.NoActive {
		dev := core.GetDevice(ob.Id)
		// 设备在其它节点时转发给其它节点执行
		if cluster.Enabled() && dev != nil && len(dev.ClusterId) > 0 && dev.ClusterId != cluster.GetClusterId() {
			resp, err := cluster.SingleInvoke(dev.ClusterId, ctl.Request)
			if err != nil {
				ctl.RespError(err)
				return
			}
			ctl.RespOkClusterData(resp)
			return
		} else {
			alins.State = core.GetDeviceState(ob.Id, ob.ProductId)
			device.UpdateOnlineStatus(ob.Id, alins.State)
		}
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
	deviceId := ctl.Param("id")
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
	deviceId := ctl.Param("id")
	_, err := ctl.getDeviceAndCheckCreateId(deviceId)
	if err != nil {
		ctl.RespError(err)
		return
	}
	if cluster.Enabled() {
		if cluster.Shard(deviceId) {
			err = connectClientDevice(deviceId)
		} else {
			err = cluster.BroadcastInvoke(ctl.Request)
		}
	} else {
		err = connectClientDevice(deviceId)
	}
	if err != nil {
		ctl.RespError(err)
		return
	}
	ctl.RespOk()
}

func (ctl *DeviceController) Disconnect() {
	if ctl.isForbidden(deviceResource, SaveAction) {
		return
	}
	deviceId := ctl.Param("id")
	_, err := ctl.getDeviceAndCheckCreateId(deviceId)
	if err != nil {
		ctl.RespError(err)
		return
	}
	dev := core.GetDevice(deviceId)
	// 设备在其它节点时转发给其它节点执行
	if cluster.Enabled() && dev != nil && len(dev.ClusterId) > 0 && dev.ClusterId != cluster.GetClusterId() {
		resp, err := cluster.SingleInvoke(dev.ClusterId, ctl.Request)
		if err != nil {
			ctl.RespError(err)
			return
		}
		ctl.RespOkClusterData(resp)
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
		ctl.RespError(errors.New("设备离线"))
		return
	}
	ctl.RespOk()
}

// deploy device
func (ctl *DeviceController) Deploy() {
	if ctl.isForbidden(deviceResource, SaveAction) {
		return
	}
	deviceId := ctl.Param("id")
	ctl.enable(deviceId, true)
	ctl.RespOk()
}

// undeploy device
func (ctl *DeviceController) Undeploy() {
	if ctl.isForbidden(deviceResource, SaveAction) {
		return
	}
	deviceId := ctl.Param("id")
	ctl.enable(deviceId, false)
	ctl.RespOk()
}

// batch deploy device
func (ctl *DeviceController) BatchDeploy() {
	if ctl.isForbidden(deviceResource, SaveAction) {
		return
	}
	var deviceIds []string
	ctl.BindJSON(&deviceIds)
	ctl.batchEnable(deviceIds, core.SearchTerm{Key: "state", Value: core.NoActive, Oper: core.EQ}, core.OFFLINE)
}

// batch undeploy device
func (ctl *DeviceController) BatchUndeploy() {
	if ctl.isForbidden(deviceResource, SaveAction) {
		return
	}
	var deviceIds []string
	ctl.BindJSON(&deviceIds)
	ctl.batchEnable(deviceIds, core.SearchTerm{Key: "state", Value: core.NoActive, Oper: core.NEQ}, core.NoActive)
}

func (ctl *DeviceController) batchEnable(deviceIds []string, term core.SearchTerm, tagertState string) {
	token := fmt.Sprintf("batch-%s-device-%v", tagertState, time.Now().UnixMicro())
	setSseData(token, "")
	isDeploy := true
	if tagertState == core.NoActive {
		isDeploy = false
	}
	go func() {
		total := 0
		resp := `{"success":true, "result": {"finish": %v, "num": %d}}`
		if len(deviceIds) > 0 {
			for _, deviceId := range deviceIds {
				ctl.enable(deviceId, isDeploy)
				total = total + 1
				if total%5 == 0 {
					setSseData(token, fmt.Sprintf(resp, false, total))
				}
			}
			setSseData(token, fmt.Sprintf(resp, true, total))
		} else {
			condition := []core.SearchTerm{}
			condition = append(condition, term)
			for {
				var page *models.PageQuery = &models.PageQuery{PageSize: 500, PageNum: 1, Condition: condition}
				result, err := device.PageDevice(page, &ctl.GetCurrentUser().Id)
				if err != nil {
					logs.Errorf("batch enable error: %v", err)
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
						devopr = dev.ToDeviceOper()
					}
					model := models.DeviceModel{}
					model.FromEnitty(dev)
					devopr.Config = model.Metaconfig
					core.PutDevice(devopr)
				}
				err = device.UpdateOnlineStatusList(ids, tagertState)
				if err != nil {
					logs.Errorf("update device state error: %v", err)
				} else {
					total = total + len(list)
				}
				setSseData(token, fmt.Sprintf(resp, false, total))
				time.Sleep(time.Millisecond * 500)
			}
			setSseData(token, fmt.Sprintf(resp, true, total))
			logs.Infof("batch deploy done")
		}
	}()
	ctl.RespOkData(token)
}

// enable device
func (ctl *DeviceController) enable(deviceId string, isDeploy bool) {
	dev, err := ctl.getDeviceAndCheckCreateId(deviceId)
	if err != nil {
		ctl.RespError(err)
		return
	}
	var state string
	if isDeploy {
		if len(dev.State) == 0 || dev.State == core.NoActive {
			state = core.OFFLINE
		}
		devopr := core.GetDevice(dev.Id)
		if devopr == nil {
			devopr = dev.ToDeviceOper()
		}
		devopr.Config = dev.Metaconfig
		core.PutDevice(devopr)
	} else {
		state = core.NoActive
		core.DeleteDevice(deviceId)
	}
	if ctl.IsNotClusterRequest() {
		device.UpdateOnlineStatus(deviceId, state)
		cluster.BroadcastInvoke(ctl.Request)
	}
}

// 命令下发
func (ctl *DeviceController) CmdInvoke() {
	if ctl.isForbidden(deviceResource, SaveAction) {
		return
	}
	deviceId := ctl.Param("id")

	var ob common.FuncInvoke
	err := ctl.BindJSON(&ob)
	if err != nil {
		ctl.RespError(err)
		return
	}
	ob.DeviceId = deviceId
	_, err = ctl.getDeviceAndCheckCreateId(ob.DeviceId)
	if err != nil {
		ctl.RespError(err)
		return
	}
	device := core.GetDevice(deviceId)
	if cluster.Enabled() && device != nil && device.ClusterId != cluster.GetClusterId() {
		ctl.Request.Header.Add(cluster.X_Cluster_Timeout, "13")
		resp, err := cluster.SingleInvoke(device.ClusterId, ctl.Request)
		if err != nil {
			ctl.RespError(err)
			return
		}
		ctl.RespOkClusterData(resp)
		return
	} else {
		err1 := core.DoCmdInvoke(ob)
		if err1 != nil {
			ctl.RespErr(err1)
			return
		}
	}
	ctl.RespOk()
}

// 查询设备属性
func (ctl *DeviceController) QueryProperty() {
	ctl.queryTimeseriesData(core.TIME_TYPE_PROP)
}

func (ctl *DeviceController) QueryLogs() {
	ctl.queryTimeseriesData(core.TIME_TYPE_LOGS)
}

func (ctl *DeviceController) QueryEvent() {
	ctl.queryTimeseriesData(core.TIME_TYPE_EVENT)
}

func (ctl *DeviceController) queryTimeseriesData(typ string) {
	if ctl.isForbidden(deviceResource, QueryAction) {
		return
	}

	deviceId := ctl.Param("id")
	var param core.TimeDataSearchRequest
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
		ctl.RespError(fmt.Errorf("产品'%s'不存在, 请确保产品已发布", device.ProductId))
		return
	}
	var res map[string]interface{}
	if typ == core.TIME_TYPE_LOGS {
		res, err = product.GetTimeSeries().QueryLogs(product, param)
	} else if typ == core.TIME_TYPE_PROP {
		res, err = product.GetTimeSeries().QueryProperty(product, param)
	} else {
		eventId := ctl.Param("eventId")
		res, err = product.GetTimeSeries().QueryEvent(product, eventId, param)
	}
	if err != nil {
		ctl.RespError(err)
		return
	}
	ctl.RespOkData(res)
}

func (ctl *DeviceController) getDeviceAndCheckCreateId(deviceId string) (*models.DeviceModel, error) {
	return device.GetDeviceAndCheckCreateId(deviceId, ctl.GetCurrentUser().Id)
}
