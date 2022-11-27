package ruleengine

import (
	"encoding/json"
	"go-iot/notify"

	"github.com/beego/beego/v2/core/logs"
)

type NotifierAction struct {
	NotifyType string                 `json:"notifyType"`
	NotifierId int64                  `json:"notifierId"`
	Data       map[string]interface{} `json:"-"`
}

func NewNotifierAction(config string) (*NotifierAction, error) {
	n := &NotifierAction{}
	err := n.FromJson(config)
	if err != nil {
		return nil, err
	}
	return n, nil
}

func (a *NotifierAction) FromJson(str string) error {
	err := json.Unmarshal([]byte(str), &a)
	return err
}

func (s *NotifierAction) Do() {
	n := notify.GetNotify(s.NotifierId)
	if n == nil {
		logs.Warn("notify not found id=%s, type=%s", s.NotifierId, s.NotifyType)
	} else {
		n.Notify(n.Title(), n.ParseTemplate(s.Data))
	}
}
