package sender

import (
	"encoding/json"
	"go-iot/models"
	"go-iot/models/modelfactory"
	"go-iot/models/operates"
)

type NorthSender struct {
	CheckAgent bool
	AgentFunc  func(device operates.Device) models.JsonResp
}

func (this NorthSender) Open(byteReq []byte, deviceId string) models.JsonResp {
	var ob []models.SwitchStatus
	json.Unmarshal(byteReq, &ob)

	device, err := modelfactory.GetDevice(deviceId)
	if err != nil {
		return models.JsonResp{Success: false, Msg: err.Error()}
	} else {
		p, err := operates.GetProvider(device.Provider)
		if err != nil {
			return models.JsonResp{Success: false, Msg: err.Error()}
		} else {
			if this.CheckAgent && len(device.Agent) > 0 {
				return this.AgentFunc(device)
			} else {
				var switchOper operates.ISwitchOper
				switchOper = p.(operates.ISwitchOper)
				operResp := switchOper.Switch(ob, device)
				return models.JsonResp{Success: operResp.Success, Msg: operResp.Msg}
			}
		}
	}
}

func (this NorthSender) Light(byteReq []byte, deviceId string) models.JsonResp {
	var ob map[string]int
	json.Unmarshal(byteReq, &ob)

	value := ob["value"]
	device, err := modelfactory.GetDevice(deviceId)
	if err != nil {
		return models.JsonResp{Success: false, Msg: err.Error()}
	} else {
		p, err := operates.GetProvider(device.Provider)
		if err != nil {
			return models.JsonResp{Success: false, Msg: err.Error()}
		} else {
			if this.CheckAgent && len(device.Agent) > 0 {
				return this.AgentFunc(device)
			} else {
				var lightOper operates.ILightOper
				lightOper = p.(operates.ILightOper)
				operResp := lightOper.Light(value, device)
				return models.JsonResp{Success: operResp.Success, Msg: operResp.Msg}
			}
		}
	}

}
