package agent

import (
	"errors"

	northsender "go-iot/controllers/sender"
	"go-iot/models"
	"go-iot/models/operates"
)

var (
	northSender northsender.NorthSender
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
	}
	if !resp.Success {
		return "", errors.New(resp.Msg)
	}
	return resp.Msg, nil
}
