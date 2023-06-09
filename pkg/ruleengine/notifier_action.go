package ruleengine

import (
	"encoding/json"
	"go-iot/pkg/notify"

	logs "go-iot/pkg/logger"
)

type NotifierAction struct {
	NotifyType string                 `json:"notifyType"`
	NotifierId int64                  `json:"notifierId"`
	Data       map[string]interface{} `json:"-"`
}

func NewNotifierAction(config string, data map[string]interface{}) (*NotifierAction, error) {
	n := &NotifierAction{}
	err := n.FromJson(config)
	if err != nil {
		return nil, err
	}
	n.Data = data
	return n, nil
}

func (a *NotifierAction) FromJson(str string) error {
	err := json.Unmarshal([]byte(str), &a)
	return err
}

func (s *NotifierAction) Do() {
	n := notify.GetNotify(s.NotifierId)
	if n == nil {
		logs.Warnf("notify not found id=%s, type=%s", s.NotifierId, s.NotifyType)
	} else {
		err := n.Notify(n.ParseTemplate(s.Data))
		if err != nil {
			logs.Warnf(err.Error())
		}
	}
}
