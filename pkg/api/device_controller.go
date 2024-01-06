package api

import (
	"errors"
	"fmt"
	"go-iot/pkg/api/web"
	"go-iot/pkg/cluster"
	"go-iot/pkg/core"
	"go-iot/pkg/models"
	deviceDao "go-iot/pkg/models/device"
	networkDao "go-iot/pkg/models/network"
	"go-iot/pkg/network"
	"net/http"
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
	d := &deviceApi{}
	web.RegisterAPI("/device/page", "POST", d.Page)
	web.RegisterAPI("/device", "POST", d.Add)
	web.RegisterAPI("/device", "PUT", d.Update)
	web.RegisterAPI("/device/{id}", "DELETE", d.Delete)
	web.RegisterAPI("/device/{id}", "GET", d.GetOne)
	web.RegisterAPI("/device/{id}/detail", "GET", d.GetDetail)
	web.RegisterAPI("/device/{id}/connect", "POST", d.Connect)
	web.RegisterAPI("/device/{id}/disconnect", "POST", d.Disconnect)
	web.RegisterAPI("/device/{id}/deploy", "POST", d.Deploy)
	web.RegisterAPI("/device/{id}/undeploy", "POST", d.Undeploy)
	web.RegisterAPI("/device/batch/deploy", "POST", d.BatchDeploy)
	web.RegisterAPI("/device/batch/undeploy", "POST", d.BatchUndeploy)
	web.RegisterAPI("/device/{id}/invoke", "POST", d.CmdInvoke)
	web.RegisterAPI("/device/{id}/properties", "POST", d.QueryProperty)
	web.RegisterAPI("/device/{id}/logs", "POST", d.QueryLogs)
	web.RegisterAPI("/device/{id}/event/{eventId}", "POST", d.QueryEvent)

	RegResource(deviceResource)
}

type deviceApi struct {
}

// 查询设备列表
func (d *deviceApi) Page(w http.ResponseWriter, r *http.Request) {
	ctl := NewAuthController(w, r)

	if ctl.isForbidden(deviceResource, QueryAction) {
		return
	}
	var ob models.PageQuery
	err := ctl.BindJSON(&ob)
	if err != nil {
		ctl.RespError(err)
		return
	}

	res, err := deviceDao.PageDevice(&ob, &ctl.GetCurrentUser().Id)
	if err != nil {
		ctl.RespError(err)
		return
	}
	ctl.RespOkData(res)
}

// 查询单个设备
func (d *deviceApi) GetOne(w http.ResponseWriter, r *http.Request) {
	ctl := NewAuthController(w, r)
	if ctl.isForbidden(deviceResource, QueryAction) {
		return
	}
	deviceId := ctl.Param("id")
	ob, err := getDeviceAndCheckCreateId(ctl, deviceId)
	if err != nil {
		ctl.RespError(err)
		return
	}
	ctl.RespOkData(ob)
}

// get device detail info
func (d *deviceApi) GetDetail(w http.ResponseWriter, r *http.Request) {
	ctl := NewAuthController(w, r)
	if ctl.isForbidden(deviceResource, QueryAction) {
		return
	}
	ob, err := getDeviceAndCheckCreateId(ctl, ctl.Param("id"))
	if err != nil {
		ctl.RespError(err)
		return
	}
	product, err := deviceDao.GetProductMust(ob.ProductId)
	if err != nil {
		ctl.RespError(err)
		return
	}
	nw, err := networkDao.GetByProductId(ob.ProductId)
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
			ctl.Resp(*resp)
			return
		} else {
			alins.State = core.GetDeviceState(ob.Id, ob.ProductId)
			deviceDao.UpdateOnlineStatus(ob.Id, alins.State)
		}
	}
	ctl.RespOkData(alins)
}

// 添加设备
func (d *deviceApi) Add(w http.ResponseWriter, r *http.Request) {
	ctl := NewAuthController(w, r)
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
	err = deviceDao.AddDevice(&ob)
	if err != nil {
		ctl.RespError(err)
		return
	}
	ctl.RespOk()
}

// 更新设备信息
func (d *deviceApi) Update(w http.ResponseWriter, r *http.Request) {
	ctl := NewAuthController(w, r)
	if ctl.isForbidden(deviceResource, SaveAction) {
		return
	}
	var ob models.DeviceModel
	err := ctl.BindJSON(&ob)
	if err != nil {
		ctl.RespError(err)
		return
	}
	_, err = getDeviceAndCheckCreateId(ctl, ob.Id)
	if err != nil {
		ctl.RespError(err)
		return
	}
	en := ob.ToEnitty()
	deviceOper := ob.ToDeviceOper()
	err = core.OnDeviceUnDeploy(deviceOper)
	if err != nil {
		ctl.RespError(err)
		return
	}
	err = deviceDao.UpdateDevice(&en)
	if err != nil {
		ctl.RespError(err)
		return
	}
	ctl.RespOk()
}

// 删除设备
func (d *deviceApi) Delete(w http.ResponseWriter, r *http.Request) {
	ctl := NewAuthController(w, r)
	if ctl.isForbidden(deviceResource, DeleteAction) {
		return
	}
	deviceId := ctl.Param("id")
	_, err := getDeviceAndCheckCreateId(ctl, deviceId)
	if err != nil {
		ctl.RespError(err)
		return
	}
	err = deviceDao.DeleteDevice(deviceId)
	if err != nil {
		ctl.RespError(err)
		return
	}
	ctl.RespOk()
}

// client设备连接
func (d *deviceApi) Connect(w http.ResponseWriter, r *http.Request) {
	ctl := NewAuthController(w, r)
	if ctl.isForbidden(deviceResource, SaveAction) {
		return
	}
	deviceId := ctl.Param("id")
	_, err := getDeviceAndCheckCreateId(ctl, deviceId)
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

func (d *deviceApi) Disconnect(w http.ResponseWriter, r *http.Request) {
	ctl := NewAuthController(w, r)
	if ctl.isForbidden(deviceResource, SaveAction) {
		return
	}
	deviceId := ctl.Param("id")
	_, err := getDeviceAndCheckCreateId(ctl, deviceId)
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
		ctl.Resp(*resp)
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
func (d *deviceApi) Deploy(w http.ResponseWriter, r *http.Request) {
	ctl := NewAuthController(w, r)
	if ctl.isForbidden(deviceResource, SaveAction) {
		return
	}
	deviceId := ctl.Param("id")
	enableDevice(ctl, deviceId, true)
}

// undeploy device
func (d *deviceApi) Undeploy(w http.ResponseWriter, r *http.Request) {
	ctl := NewAuthController(w, r)
	if ctl.isForbidden(deviceResource, SaveAction) {
		return
	}
	deviceId := ctl.Param("id")
	enableDevice(ctl, deviceId, false)
}

// batch deploy device
func (d *deviceApi) BatchDeploy(w http.ResponseWriter, r *http.Request) {
	ctl := NewAuthController(w, r)
	if ctl.isForbidden(deviceResource, SaveAction) {
		return
	}
	var deviceIds []string
	ctl.BindJSON(&deviceIds)
	batchEnableDevice(ctl, deviceIds, core.SearchTerm{Key: "state", Value: core.NoActive, Oper: core.EQ}, core.OFFLINE)
}

// batch undeploy device
func (d *deviceApi) BatchUndeploy(w http.ResponseWriter, r *http.Request) {
	ctl := NewAuthController(w, r)
	if ctl.isForbidden(deviceResource, SaveAction) {
		return
	}
	var deviceIds []string
	ctl.BindJSON(&deviceIds)
	batchEnableDevice(ctl, deviceIds, core.SearchTerm{Key: "state", Value: core.NoActive, Oper: core.NEQ}, core.NoActive)
}

// 命令下发
func (d *deviceApi) CmdInvoke(w http.ResponseWriter, r *http.Request) {
	ctl := NewAuthController(w, r)
	if ctl.isForbidden(deviceResource, SaveAction) {
		return
	}
	deviceId := ctl.Param("id")

	var ob core.FuncInvoke
	err := ctl.BindJSON(&ob)
	if err != nil {
		ctl.RespError(err)
		return
	}
	ob.DeviceId = deviceId
	_, err = getDeviceAndCheckCreateId(ctl, deviceId)
	if err != nil {
		ctl.RespError(err)
		return
	}
	deviceOper := core.GetDevice(deviceId)
	if deviceOper == nil {
		ctl.RespError(errors.New("设备未激活"))
		return
	}
	sendCluster := cluster.Enabled() && deviceOper.ClusterId != cluster.GetClusterId()
	productOper := core.GetProduct(deviceOper.ProductId)
	if productOper == nil {
		ctl.RespError(fmt.Errorf("产品'%s'不存在或未发布", deviceOper.ProductId))
		return
	}
	// 无状态网络不用走集群
	if network.IsStateless(productOper.NetworkType) {
		sendCluster = false
	}
	if sendCluster {
		ctl.Request.Header.Add(cluster.X_Cluster_Timeout, "13")
		resp, err := cluster.SingleInvoke(deviceOper.ClusterId, ctl.Request)
		if err != nil {
			ctl.RespError(err)
			return
		}
		ctl.Resp(*resp)
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
func (d *deviceApi) QueryProperty(w http.ResponseWriter, r *http.Request) {
	ctl := NewAuthController(w, r)
	queryDeviceTimeseriesData(ctl, core.TIME_TYPE_PROP)
}

func (d *deviceApi) QueryLogs(w http.ResponseWriter, r *http.Request) {
	ctl := NewAuthController(w, r)
	queryDeviceTimeseriesData(ctl, core.TIME_TYPE_LOGS)
}

func (d *deviceApi) QueryEvent(w http.ResponseWriter, r *http.Request) {
	ctl := NewAuthController(w, r)
	queryDeviceTimeseriesData(ctl, core.TIME_TYPE_EVENT)
}

// 批量启用、禁用设备
func batchEnableDevice(ctl *AuthController, deviceIds []string, term core.SearchTerm, tagertState string) {
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
				enableDevice(ctl, deviceId, isDeploy)
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
				result, err := deviceDao.PageDevice(page, &ctl.GetCurrentUser().Id)
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
				err = deviceDao.UpdateOnlineStatusList(ids, tagertState)
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

// 启动、禁用设备
func enableDevice(ctl *AuthController, deviceId string, isDeploy bool) {
	dev, err := getDeviceAndCheckCreateId(ctl, deviceId)
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
		err = core.OnDeviceDeploy(devopr)
		if err != nil {
			ctl.RespError(err)
			return
		}
		// OnDeviceDeploy可以设置Config
		if len(devopr.Config) > 0 {
			if dev.Metaconfig == nil {
				dev.Metaconfig = map[string]string{}
			}
			for key, v := range devopr.Config {
				dev.Metaconfig[key] = v
			}
			entity := dev.ToEnitty()
			deviceDao.UpdateDevice(&entity)
		}
		devopr.Config = dev.Metaconfig
		core.PutDevice(devopr)
	} else {
		devopr := core.GetDevice(deviceId)
		if devopr == nil {
			ctl.RespError(errors.New("设备未激活"))
			return
		}
		err = core.OnDeviceUnDeploy(devopr)
		if err != nil {
			ctl.RespError(err)
			return
		}
		state = core.NoActive
		core.DeleteDevice(deviceId)
	}
	if ctl.IsNotClusterRequest() {
		deviceDao.UpdateOnlineStatus(deviceId, state)
		cluster.BroadcastInvoke(ctl.Request)
	}
	ctl.RespOk()
}

// 查询设备时序数据
func queryDeviceTimeseriesData(ctl *AuthController, typ string) {
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
	device, err := getDeviceAndCheckCreateId(ctl, deviceId)
	if err != nil {
		ctl.RespError(err)
		return
	}
	productOper := core.GetProduct(device.ProductId)
	if productOper == nil {
		ctl.RespError(fmt.Errorf("产品'%s'不存在或未发布", device.ProductId))
		return
	}
	var res map[string]interface{}
	if typ == core.TIME_TYPE_LOGS {
		res, err = productOper.GetTimeSeries().QueryLogs(productOper, param)
	} else if typ == core.TIME_TYPE_PROP {
		res, err = productOper.GetTimeSeries().QueryProperty(productOper, param)
	} else {
		eventId := ctl.Param("eventId")
		res, err = productOper.GetTimeSeries().QueryEvent(productOper, eventId, param)
	}
	if err != nil {
		ctl.RespError(err)
		return
	}
	ctl.RespOkData(res)
}

func getDeviceAndCheckCreateId(ctl *AuthController, deviceId string) (*models.DeviceModel, error) {
	return deviceDao.GetDeviceAndCheckCreateId(deviceId, ctl.GetCurrentUser().Id)
}
