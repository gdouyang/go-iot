package sender

import (
	"encoding/json"
	"go-iot/models"
	"go-iot/models/modelfactory"
	"go-iot/models/operates"
)

const (
	// 开关
	OPER_OPEN = "open"
	// 调光
	OPER_LIGHT = "light"
	//
	OPER_GET_ONLINESTATUS = "getOnlineStatus"
)

func init() {
	// northSender := NorthSender{}
	// agent.RegProcessFunc(OPER_OPEN, func(request agent.AgentRequest) models.JsonResp {
	// 	res := northSender.Open(request.Data, request.DeviceId)
	// 	return res
	// })

	// agent.RegProcessFunc(OPER_LIGHT, func(request agent.AgentRequest) models.JsonResp {
	// 	res := northSender.Light(request.Data, request.DeviceId)
	// 	return res
	// })
	// agent.RegProcessFunc(OPER_GET_ONLINESTATUS, func(request agent.AgentRequest) models.JsonResp {
	// 	res := northSender.GetOnlineStatus(request.Data, request.DeviceId)
	// 	return res
	// })
}

type NorthSender struct {
	// 是否检查有Agent
	CheckAgent bool
}

// 当设备通过Agent上线时执行此方法，把命令下发给Agent让Agent再下发给设备
func (this NorthSender) SendAgent(device operates.Device, oper string, data models.IotRequest) models.JsonResp {
	// req := agent.NewRequest(device.Id, device.Sn, device.Provider, oper, data)
	// res := agent.SendCommand(device.Agent, req)
	return models.JsonResp{}
}

// 开关操作
func (this NorthSender) Open(iotReq models.IotRequest, deviceId string) models.JsonResp {
	echoToBrower(iotReq)
	data := iotReq.Data
	var ob []models.SwitchStatus
	json.Unmarshal(data, &ob)

	device, err := modelfactory.GetDevice(deviceId)
	if err != nil {
		return models.JsonResp{Success: false, Msg: err.Error()}
	}
	if this.CheckAgent && len(device.Agent) > 0 {
		return this.SendAgent(device, OPER_OPEN, iotReq)
	}
	p, err := operates.GetProvider(device.Provider)
	if err != nil {
		return models.JsonResp{Success: false, Msg: err.Error()}
	}
	var switchOper operates.ISwitchOper
	switchOper, ok := p.(operates.ISwitchOper)
	if !ok {
		return models.JsonResp{Success: false, Msg: "厂商没有开关功能"}
	}
	operResp := switchOper.Switch(ob, device)
	return models.JsonResp{Success: operResp.Success, Msg: operResp.Msg}
}

// 调光操作
func (this NorthSender) Light(iotReq models.IotRequest, deviceId string) models.JsonResp {
	echoToBrower(iotReq)
	data := iotReq.Data
	var ob map[string]int
	json.Unmarshal(data, &ob)

	value := ob["value"]
	device, err := modelfactory.GetDevice(deviceId)
	if err != nil {
		return models.JsonResp{Success: false, Msg: err.Error()}
	}
	if this.CheckAgent && len(device.Agent) > 0 {
		return this.SendAgent(device, OPER_LIGHT, iotReq)
	}
	p, err := operates.GetProvider(device.Provider)
	if err != nil {
		return models.JsonResp{Success: false, Msg: err.Error()}
	}
	var lightOper operates.ILightOper
	lightOper, ok := p.(operates.ILightOper)
	if !ok {
		return models.JsonResp{Success: false, Msg: "厂商没有调光功能"}
	}
	operResp := lightOper.Light(value, device)
	return models.JsonResp{Success: operResp.Success, Msg: operResp.Msg}
}

// 获取在线状态
func (this NorthSender) GetOnlineStatus(iotReq models.IotRequest, deviceId string) models.JsonResp {
	echoToBrower(iotReq)
	device, err := modelfactory.GetDevice(deviceId)
	if err != nil {
		return models.JsonResp{Success: false, Msg: err.Error()}
	}
	status := models.OFFLINE
	defer func() {
		// 更新在线状态
		evt := operates.DeviceOnlineStatus{OnlineStatus: status, Sn: device.Sn, Provider: device.Provider}
		modelfactory.FireOnlineStatus(evt)
	}()
	if this.CheckAgent && len(device.Agent) > 0 {
		resp := this.SendAgent(device, OPER_GET_ONLINESTATUS, iotReq)
		if len(resp.Data) > 0 {
			status = string(resp.Data)
		}
		return resp
	}
	p, err := operates.GetProvider(device.Provider)
	if err != nil {
		return models.JsonResp{Success: false, Msg: err.Error()}
	}
	var oper operates.IOnlineStatusOper
	oper, ok := p.(operates.IOnlineStatusOper)
	if !ok {
		return models.JsonResp{Success: false, Msg: "厂商没有获取在线状态功能"}
	}
	status = oper.GetOnlineStatus(device)
	return models.JsonResp{Success: true, Msg: status, Data: []byte(status)}
}
