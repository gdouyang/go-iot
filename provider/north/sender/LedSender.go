package sender

import (
	"encoding/json"
	"go-iot/agent"
	"go-iot/models"
	"go-iot/models/led"
)

const (
	LED_ADD    = "ledAdd"
	LED_UPDATE = "ledUpdate"
	LED_DELETE = "ledDelete"
)

func init() {
	ledSender := LedSender{}
	agent.RegProcessFunc(LED_ADD, func(request agent.AgentRequest) models.JsonResp {
		res := ledSender.Add(request.Data)
		return res
	})

	agent.RegProcessFunc(LED_UPDATE, func(request agent.AgentRequest) models.JsonResp {
		res := ledSender.Update(request.Data)
		return res
	})

	agent.RegProcessFunc(LED_DELETE, func(request agent.AgentRequest) models.JsonResp {
		res := ledSender.Delete(request.Data)
		return res
	})
}

type LedSender struct {
	// 是否检查有Agent
	CheckAgent bool
}

// 当设备通过Agent上线时执行此方法，把命令下发给Agent让Agent再下发给设备
func (this LedSender) SendAgent(device led.Device, oper string, data []byte) models.JsonResp {
	req := agent.NewRequest(device.Id, device.Sn, device.Provider, oper, data)
	res := agent.SendCommand(device.Agent, req)
	return res
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
		aResp := this.SendAgent(ob, LED_ADD, data)
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
		aResp := this.SendAgent(ob, LED_UPDATE, data)
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
		aResp := this.SendAgent(ob, LED_DELETE, data)
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