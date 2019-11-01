package agent

import (
	"errors"

	"go-iot/models"
)

var (
	processMap map[string]func(request AgentRequest) models.JsonResp = map[string]func(request AgentRequest) models.JsonResp{}
)

// 注册Agent处理函数，当控制层下发命令给Agent时Agent把相应指令发给具体厂商设备执行
// oper 操作项
// process 处理函数
func RegProcessFunc(oper string, process func(request AgentRequest) models.JsonResp) {
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
		return "", errors.New("Agent[" + resp.Msg + "]")
	}
	return resp.Msg, nil
}
