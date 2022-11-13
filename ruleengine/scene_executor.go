package ruleengine

import (
	"encoding/json"
	"fmt"
	"go-iot/codec/eventbus"
)

type SceneExecutor struct {
	Triggers []SceneTrigger
	Actions  []Action
}

func (s *SceneExecutor) Init() {
	eventbus.Subscribe(fmt.Sprintf(eventbus.DeviceMessageTopic, "*", "*"), func(data interface{}) {
		s.doStart(data)
	})
}

func (s *SceneExecutor) doStart(data interface{}) {
	var pass bool
	for _, trigger := range s.Triggers {
		device := trigger.Device
		for _, filter := range device.Filters {
			pass, _ = filter.evaluate(data.(map[string]interface{}))
			goto here
		}
	}
here:
	if pass {
		for _, action := range s.Actions {
			if action.Executor == "device-message-sender" {
				a := &DeviceCmdAction{}
				b, _ := json.Marshal(action.Configuration)
				a.FromJson(string(b))
				a.Do()
			}
		}
	}
}
