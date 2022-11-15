package ruleengine

import (
	"encoding/json"
	"go-iot/codec"
	"go-iot/codec/msg"
)

type DeviceCmdAction struct {
	message msg.FuncInvoke
}

func (a *DeviceCmdAction) Covent(config interface{}) error {
	b, _ := json.Marshal(config)
	err := a.FromJson(string(b))
	return err
}

func (a *DeviceCmdAction) FromJson(str string) error {
	err := json.Unmarshal([]byte(str), &a.message)
	return err
}

func (s *DeviceCmdAction) Do() {
	codec.DoCmdInvoke("", s.message)
}
