package sender

import (
	"encoding/json"
	"fmt"
	"go-iot/agent"
	"go-iot/models"
	"go-iot/models/modelfactory"
	"go-iot/models/operates"
	xixun "go-iot/provider/xixun/base"
	"strings"

	"go-iot/provider/util"
)

const (
	SCREEN_SHOT = "xixunScreenShot"
	MSG_CLEAR   = "xixunMsgClear"
	FILE_UPLOAD = "xixunFileUpload"
	MSG_PUBLISH = "xixunMsgPublish"
	LED_PLAY    = "xixunLedPlay"
)

func init() {
	xixunSender := XixunSender{}
	agent.RegProcessFunc(SCREEN_SHOT, func(request agent.AgentRequest) models.JsonResp {
		res := xixunSender.ScreenShot(request.DeviceId)
		return res
	})

	agent.RegProcessFunc(MSG_CLEAR, func(request agent.AgentRequest) models.JsonResp {
		res := xixunSender.ClearScreenText(request.DeviceId)
		return res
	})

	agent.RegProcessFunc(MSG_PUBLISH, func(request agent.AgentRequest) models.JsonResp {
		res := xixunSender.MsgPublish(request.Data, request.DeviceId)
		return res
	})
	agent.RegProcessFunc(FILE_UPLOAD, func(request agent.AgentRequest) models.JsonResp {
		res := xixunSender.FileUpload(request.Data, request.DeviceId)
		return res
	})
	agent.RegProcessFunc(LED_PLAY, func(request agent.AgentRequest) models.JsonResp {
		res := xixunSender.LedPlay(request.Data, request.DeviceId)
		return res
	})
}

type XixunSender struct {
	// 是否检查有Agent
	CheckAgent bool
}

// 当设备通过Agent上线时执行此方法，把命令下发给Agent让Agent再下发给设备
func (this XixunSender) SendAgent(device operates.Device, oper string, data []byte) models.JsonResp {
	req := agent.NewRequest(device.Id, device.Sn, device.Provider, oper, data)
	res := agent.SendCommand(device.Agent, req)
	return res
}

// LED截图
func (this XixunSender) ScreenShot(deviceId string) models.JsonResp {
	device, err := modelfactory.GetDevice(deviceId)
	if err != nil {
		return models.JsonResp{Success: false, Msg: err.Error()}
	}

	if this.CheckAgent && len(device.Agent) > 0 {
		return this.SendAgent(device, SCREEN_SHOT, []byte("{}"))
	}

	operResp := xixun.ProviderImplXiXunLed.ScreenShot(device.Sn)
	return models.JsonResp{Success: operResp.Success, Msg: operResp.Msg}
}

// LED播放文件上传
func (this XixunSender) FileUpload(data []byte, deviceId string) models.JsonResp {
	device, err := modelfactory.GetDevice(deviceId)
	if err != nil {
		return models.JsonResp{Success: false, Msg: err.Error()}
	}
	if this.CheckAgent && len(device.Agent) > 0 {
		return this.SendAgent(device, FILE_UPLOAD, data)
	}
	var param map[string]string
	json.Unmarshal(data, &param)
	paths := param["paths"]
	materialPaths := strings.Split(paths, ",")
	serverUrl := param["serverUrl"]
	serverUrl += "/file/"
	msg := ""
	for _, path := range materialPaths {
		operResp := xixun.ProviderImplXiXunLed.FileUpload(device.Sn, serverUrl+path, path)
		msg += operResp.Msg
	}
	return models.JsonResp{Success: true, Msg: msg}
}

/*
播放，播放zip文件、MP4播放素材、rtsp视频流
业务1：制定MP4播放素材列表，并点播 (待定)
业务2：查看内部存储里面zip文件是否存在，不存在则调用文件下发，然后再发起播放
*/
func (this XixunSender) LedPlay(data []byte, deviceId string) models.JsonResp {
	device, err := modelfactory.GetDevice(deviceId)
	if err != nil {
		return models.JsonResp{Success: false, Msg: err.Error()}
	}
	if this.CheckAgent && len(device.Agent) > 0 {
		return this.SendAgent(device, LED_PLAY, data)
	}
	var param map[string]string
	json.Unmarshal(data, &param)
	paths := param["paths"]
	serverUrl := param["serverUrl"]
	serverUrl += "/file/"
	filename := paths
	// 查看文件长度，并与远程对比
	fileSize := util.FileSize("./files/" + paths)
	leg, err := xixun.ProviderImplXiXunLed.FileLength(filename, device.Sn)
	if err != nil {
		return models.JsonResp{Success: false, Msg: err.Error()}
	}
	if fileSize != leg {
		//长度不一致，则返回，让重新上传
		return models.JsonResp{Success: false, Msg: fmt.Sprintf("文件长度不一致，文件没有上传成功 %s %s", fileSize, leg)}
	}
	//如果长度一致，就发起播放
	operResp := xixun.ProviderImplXiXunLed.PlayZip(filename, device.Sn)
	return models.JsonResp{Success: operResp.Success, Msg: operResp.Msg}
}

// 发布消息
func (this XixunSender) MsgPublish(data []byte, deviceId string) models.JsonResp {
	device, err := modelfactory.GetDevice(deviceId)
	if err != nil {
		return models.JsonResp{Success: false, Msg: err.Error()}
	}

	if this.CheckAgent && len(device.Agent) > 0 {
		return this.SendAgent(device, MSG_PUBLISH, data)
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
		return this.SendAgent(device, MSG_CLEAR, []byte("{}"))
	}

	operResp := xixun.ProviderImplXiXunLed.Clear(device.Sn)
	return models.JsonResp{Success: operResp.Success, Msg: operResp.Msg}
}
