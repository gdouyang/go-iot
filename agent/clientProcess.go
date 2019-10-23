package agent

import (
	"encoding/json"
	"errors"
	"go-iot/models"
	"go-iot/models/modelfactory"
	"go-iot/models/operates"
)

func processRequest(request AgentRequest) (string, error) {
	if len(request.Provider) == 0 {
		return "", errors.New("厂商不能为空")
	}
	var resp models.JsonResp
	if request.Oper == operates.OPER_OPEN {
		resp = Open(request.Data, request.DeviceId)
	} else if request.Oper == operates.OPER_LIGHT {
		resp = Light(request.Data, request.DeviceId)
	}
	if !resp.Success {
		return "", errors.New(resp.Msg)
	}
	return resp.Msg, nil
}

func Open(byteReq []byte, deviceId string) models.JsonResp {
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
			var switchOper operates.ISwitchOper
			switchOper = p.(operates.ISwitchOper)
			operResp := switchOper.Switch(ob, device)
			return models.JsonResp{Success: operResp.Success, Msg: operResp.Msg}
		}
	}
}

func Light(byteReq []byte, deviceId string) models.JsonResp {
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
			var lightOper operates.ILightOper
			lightOper = p.(operates.ILightOper)
			operResp := lightOper.Light(value, device)
			return models.JsonResp{Success: operResp.Success, Msg: operResp.Msg}
		}
	}

}
