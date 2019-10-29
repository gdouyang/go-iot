package sender

import (
	"encoding/json"
	"go-iot/models"
	"go-iot/models/led"
)

var (
	LED_ADD    = "ledAdd"
	LED_UPDATE = "ledUpdate"
	LED_DELETE = "ledDelete"
)

type LedSender struct {
	// 是否检查有Agent
	CheckAgent bool
	// 当设备通过Agent上线时执行此方法，把命令下发给Agent让Agent再下发给设备
	AgentFunc func(device led.Device) models.JsonResp
}

func (this LedSender) Add(data []byte) models.JsonResp {
	var ob led.Device
	err := json.Unmarshal(data, &ob)
	if err != nil {
		return models.JsonResp{Success: false, Msg: err.Error()}
	}
	var resp models.JsonResp
	err = led.AddDevie(&ob)
	resp.Success = true
	resp.Msg = "添加成功!"
	if err != nil {
		return models.JsonResp{Success: false, Msg: err.Error()}
	}
	if this.CheckAgent && len(ob.Agent) > 0 {
		aResp := this.AgentFunc(ob)
		if !aResp.Success {
			return aResp
		}
	}
	return resp
}

func (this LedSender) Update(data []byte) models.JsonResp {
	var ob led.Device
	err := json.Unmarshal(data, &ob)
	if err != nil {
		return models.JsonResp{Success: false, Msg: err.Error()}
	}

	err = led.UpdateDevice(&ob)
	var resp models.JsonResp
	resp.Success = true
	resp.Msg = "修改成功!"
	if err != nil {
		return models.JsonResp{Success: false, Msg: err.Error()}
	}
	if this.CheckAgent && len(ob.Agent) > 0 {
		aResp := this.AgentFunc(ob)
		if !aResp.Success {
			return aResp
		}
	}
	return resp
}

func (this LedSender) Delete(data []byte) models.JsonResp {
	var ob led.Device
	err := json.Unmarshal(data, &ob)
	if err != nil {
		return models.JsonResp{Success: false, Msg: err.Error()}
	}

	if this.CheckAgent && len(ob.Agent) > 0 {
		aResp := this.AgentFunc(ob)
		if !aResp.Success {
			return aResp
		}
	}
	err = led.DeleteDevice(&ob)
	var resp models.JsonResp
	resp.Success = true
	resp.Msg = "删除成功!"
	if err != nil {
		return models.JsonResp{Success: false, Msg: err.Error()}
	}

	return resp
}
