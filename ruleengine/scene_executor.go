package ruleengine

import (
	"go-iot/codec/eventbus"

	"github.com/beego/beego/v2/core/logs"
)

type SceneExecutor struct {
	Triggers []SceneTrigger
	Actions  []Action
}

func (s *SceneExecutor) Init() {
	for _, trigger := range s.Triggers {
		if trigger.Type == TriggerTypeDevice {
			device := trigger.Device
			eventbus.Subscribe(eventbus.GetDeviceMesssageTopic(device.ProductId, device.DeviceId), func(data interface{}) {
				s.doStart(device, data)
			})
		}
	}
}

func (s *SceneExecutor) doStart(device SceneTriggerDevice, data interface{}) {
	var pass bool
	for _, filter := range device.Filters {
		pass, _ = filter.evaluate(data.(map[string]interface{}))
		break
	}
	if pass {
		for _, action := range s.Actions {
			if action.Executor == "device-message-sender" {
				a := DeviceCmdAction{}
				err := a.Covent(action.Configuration)
				if err != nil {
					logs.Error(err)
				} else {
					a.Do()
				}
			}
		}
	}
}
