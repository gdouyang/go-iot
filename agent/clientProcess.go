package agent

import (
	"errors"

	northsender "go-iot/controllers/sender"
	"go-iot/models"
	"go-iot/models/operates"
	xixunsender "go-iot/provider/xixun/sender"
)

var (
	northSender northsender.NorthSender
	ledSender   northsender.LedSender
	xixunSender xixunsender.XixunSender
)

func processRequest(request AgentRequest) (string, error) {
	if len(request.Provider) == 0 {
		return "", errors.New("厂商不能为空")
	}
	var resp models.JsonResp
	if request.Oper == operates.OPER_OPEN {
		resp = northSender.Open(request.Data, request.DeviceId)
	} else if request.Oper == operates.OPER_LIGHT {
		resp = northSender.Light(request.Data, request.DeviceId)
	} else if request.Oper == xixunsender.SCREEN_SHOT {
		resp = xixunSender.ScreenShot(request.DeviceId)
	} else if request.Oper == xixunsender.MSG_CLEAR {
		resp = xixunSender.ClearScreenText(request.DeviceId)
	} else if request.Oper == xixunsender.MSG_PUBLISH {
		resp = xixunSender.MsgPublish(request.Data, request.DeviceId)
	} else if request.Oper == northsender.LED_ADD {
		resp = ledSender.Add(request.Data)
	} else if request.Oper == northsender.LED_UPDATE {
		resp = ledSender.Update(request.Data)
	}
	if !resp.Success {
		return "", errors.New("Agent" + resp.Msg)
	}
	return resp.Msg, nil
}
