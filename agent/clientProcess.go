package agent

import (
	"errors"

	"go-iot/models"
)

var (
	processMap map[string]func(request AgentRequest) models.JsonResp = map[string]func(request AgentRequest) models.JsonResp{}
)

func RegProcessMap(oper string, process func(request AgentRequest) models.JsonResp) {
	processMap[oper] = process
}

func processRequest(request AgentRequest) (string, error) {
	if len(request.Provider) == 0 {
		return "", errors.New("厂商不能为空")
	}
	processFunc, ok := processMap[request.Oper]
	var resp models.JsonResp
	if ok {
		resp = processFunc(request)
	}

	if !resp.Success {
		return "", errors.New("Agent" + resp.Msg)
	}
	return resp.Msg, nil
}
