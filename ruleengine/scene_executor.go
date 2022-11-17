package ruleengine

import (
	"go-iot/codec/eventbus"
	"sync"

	"github.com/beego/beego/v2/core/logs"
	"github.com/robfig/cron"
)

var manager = &sceneManager{
	m: map[int64]*SceneExecutor{},
}

type sceneManager struct {
	sync.Mutex
	m map[int64]*SceneExecutor
}

func StartScene(id int64, trigger SceneTrigger, actions []Action) {
	manager.Lock()
	defer manager.Unlock()
	e := &SceneExecutor{
		Trigger: trigger,
		Actions: actions,
	}
	e.start()
	manager.m[id] = e
}

func StopScene(id int64) {
	manager.Lock()
	defer manager.Unlock()
	if e, ok := manager.m[id]; ok {
		e.stop()
	}
}

type SceneExecutor struct {
	Id      int64
	Trigger SceneTrigger
	Actions []Action
	cron    *cron.Cron
}

func (s *SceneExecutor) start() {
	if s.Trigger.TriggerType == TriggerTypeDevice {
		device := s.Trigger
		eventbus.Subscribe(eventbus.GetDeviceMesssageTopic(device.ProductId, device.DeviceId), s.subscribeEvent)
	} else if s.Trigger.TriggerType == TriggerTypeTimer {
		go s.runCron()
	}
}

func (s *SceneExecutor) stop() {
	if s.Trigger.TriggerType == TriggerTypeDevice {
		device := s.Trigger
		eventbus.UnSubscribe(eventbus.GetDeviceMesssageTopic(device.ProductId, device.DeviceId), s.subscribeEvent)
	} else if s.Trigger.TriggerType == TriggerTypeTimer {
		if s.cron != nil {
			s.cron.Stop()
		}
	}
}

func (s *SceneExecutor) subscribeEvent(data interface{}) {
	s.doStart(s.Trigger, data)
}

func (s *SceneExecutor) doStart(device SceneTrigger, data interface{}) {
	var pass bool
	for _, filter := range device.Trigger.Filters {
		pass, _ = filter.evaluate(data.(map[string]interface{}))
		break
	}
	if pass {
		s.runAction()
	}
}

func (s *SceneExecutor) runCron() {
	s.cron = cron.New()
	s.cron.AddFunc(s.Trigger.Cron, func() {
		s.runAction()
	})
	s.cron.Start()
	defer s.cron.Stop()
	select {}
}

func (s *SceneExecutor) runAction() {
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
