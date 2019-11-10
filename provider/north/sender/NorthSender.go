package sender

import (
	"encoding/json"
	"go-iot/agent"
	"go-iot/models"
	"go-iot/models/modelfactory"
	"go-iot/models/operates"
)

func init() {
	northSender := NorthSender{}
	agent.RegProcessFunc(operates.OPER_OPEN, func(request agent.AgentRequest) models.JsonResp {
		res := northSender.Open(request.Data, request.DeviceId)
		return res
	})

	agent.RegProcessFunc(operates.OPER_LIGHT, func(request agent.AgentRequest) models.JsonResp {
		res := northSender.Light(request.Data, request.DeviceId)
		return res
	})
}

type NorthSender struct {
	// 是否检查有Agent
	CheckAgent bool
}

// 当设备通过Agent上线时执行此方法，把命令下发给Agent让Agent再下发给设备
func (this NorthSender) SendAgent(device operates.Device, oper string, data []byte) models.JsonResp {
	req := agent.NewRequest(device.Id, device.Sn, device.Provider, oper, data)
	res := agent.SendCommand(device.Agent, req)
	return res
}

// 开关操作
func (this NorthSender) Open(data []byte, deviceId string) models.JsonResp {
	var ob []models.SwitchStatus
	json.Unmarshal(data, &ob)

	device, err := modelfactory.GetDevice(deviceId)
	if err != nil {
		return models.JsonResp{Success: false, Msg: err.Error()}
	}
	if this.CheckAgent && len(device.Agent) > 0 {
		return this.SendAgent(device, operates.OPER_OPEN, data)
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
func (this NorthSender) Light(data []byte, deviceId string) models.JsonResp {
	var ob map[string]int
	json.Unmarshal(data, &ob)

	value := ob["value"]
	device, err := modelfactory.GetDevice(deviceId)
	if err != nil {
		return models.JsonResp{Success: false, Msg: err.Error()}
	}
	if this.CheckAgent && len(device.Agent) > 0 {
		return this.SendAgent(device, operates.OPER_LIGHT, data)
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
