package ruleengine

import (
	"encoding/json"
	"go-iot/pkg/core"
)

type DeviceCmdAction struct {
	message core.FuncInvoke
}

func NewDeviceCmdAction(config string) (*DeviceCmdAction, error) {
	a := &DeviceCmdAction{}
	err := a.FromJson(config)
	if err != nil {
		return nil, err
	}
	return a, nil
}

func (a *DeviceCmdAction) FromJson(str string) error {
	err := json.Unmarshal([]byte(str), &a.message)
	return err
}

func (s *DeviceCmdAction) Do() {
	core.DoCmdInvokeCluster(s.message)
}
