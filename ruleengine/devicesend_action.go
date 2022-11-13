package ruleengine

import (
	"encoding/json"
	"go-iot/codec"
	"go-iot/codec/msg"
)

type DeviceCmdAction struct {
	message msg.FuncInvoke
}

func (a *DeviceCmdAction) FromJson(str string) error {
	err := json.Unmarshal([]byte(str), &a.message)
	return err
}

func (s *DeviceCmdAction) Do() {
	codec.DoCmdInvoke("", s.message)
}
