package sender

import (
	"encoding/json"
	"go-iot/agent"
	"go-iot/models"
	"go-iot/models/modelfactory"
	"go-iot/models/operates"
	"go-iot/provider/xixun"
)

const (
	SCREEN_SHOT = "xixunScreenShot"
	MSG_CLEAR   = "xixunMsgClear"
	MSG_PUBLISH = "xixunMsgPublish"
)

func init() {
	xixunSender := XixunSender{}
	agent.RegProcessMap(SCREEN_SHOT, func(request agent.AgentRequest) models.JsonResp {
		res := xixunSender.ScreenShot(request.DeviceId)
		return res
	})

	agent.RegProcessMap(MSG_CLEAR, func(request agent.AgentRequest) models.JsonResp {
		res := xixunSender.ClearScreenText(request.DeviceId)
		return res
	})

	agent.RegProcessMap(MSG_PUBLISH, func(request agent.AgentRequest) models.JsonResp {
		res := xixunSender.MsgPublish(request.Data, request.DeviceId)
		return res
	})
}

type XixunSender struct {
	// 是否检查有Agent
	CheckAgent bool
	// 当设备通过Agent上线时执行此方法，把命令下发给Agent让Agent再下发给设备
	AgentFunc func(device operates.Device) models.JsonResp
}

// LED截图
func (this XixunSender) ScreenShot(deviceId string) models.JsonResp {
	device, err := modelfactory.GetDevice(deviceId)
	if err != nil {
		return models.JsonResp{Success: false, Msg: err.Error()}
	}

	if this.CheckAgent && len(device.Agent) > 0 {
		return this.AgentFunc(device)
	}

	operResp := xixun.ProviderImplXiXunLed.ScreenShot(device.Sn)
	return models.JsonResp{Success: operResp.Success, Msg: operResp.Msg}
}

// 发布消息
func (this XixunSender) MsgPublish(data []byte, deviceId string) models.JsonResp {
	device, err := modelfactory.GetDevice(deviceId)
	if err != nil {
		return models.JsonResp{Success: false, Msg: err.Error()}
	}

	if this.CheckAgent && len(device.Agent) > 0 {
		return this.AgentFunc(device)
	}
	param := xixun.MsgParam{}
	err = json.Unmarshal(data, &param)
	if err != nil {
		return models.JsonResp{Success: false, Msg: err.Error()}
	}
	operResp := xixun.ProviderImplXiXunLed.MsgPublish(device.Sn, param)
	return models.JsonResp{Success: operResp.Success, Msg: operResp.Msg}
}

/*清除本机的消息*/
func (this XixunSender) ClearScreenText(deviceId string) models.JsonResp {
	device, err := modelfactory.GetDevice(deviceId)
	if err != nil {
		return models.JsonResp{Success: false, Msg: err.Error()}
	}

	if this.CheckAgent && len(device.Agent) > 0 {
		return this.AgentFunc(device)
	}

	operResp := xixun.ProviderImplXiXunLed.Clear(device.Sn)
	return models.JsonResp{Success: operResp.Success, Msg: operResp.Msg}
}
