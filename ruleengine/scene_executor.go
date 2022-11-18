package ruleengine

import (
	"go-iot/codec/eventbus"
	"sync"

	"github.com/beego/beego/v2/core/logs"
	"github.com/robfig/cron/v3"
)

var manager = &sceneManager{
	m: map[int64]*SceneExecutor{},
}

var cronManager = cron.New()

func init() {
	go func() {
		cronManager.Start()
		defer cronManager.Stop()
		select {}
	}()
}

type sceneManager struct {
	sync.Mutex
	m map[int64]*SceneExecutor
}

func StartScene(id int64, trigger SceneTrigger, actions []Action) error {
	manager.Lock()
	defer manager.Unlock()
	e := &SceneExecutor{
		Trigger: trigger,
		Actions: actions,
	}
	err := e.start()
	if err != nil {
		return err
	}
	manager.m[id] = e
	return nil
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
	cronId  cron.EntryID
}

func (s *SceneExecutor) start() error {
	if s.Trigger.TriggerType == TriggerTypeDevice {
		device := s.Trigger
		eventbus.Subscribe(eventbus.GetDeviceMesssageTopic(device.ProductId, device.DeviceId), s.subscribeEvent)
		return nil
	} else if s.Trigger.TriggerType == TriggerTypeTimer {
		entryID, err := cronManager.AddFunc(s.Trigger.Cron, s.runAction)
		if err != nil {
			return err
		}
		s.cronId = entryID
	}
	return nil
}

func (s *SceneExecutor) stop() {
	if s.Trigger.TriggerType == TriggerTypeDevice {
		device := s.Trigger
		eventbus.UnSubscribe(eventbus.GetDeviceMesssageTopic(device.ProductId, device.DeviceId), s.subscribeEvent)
	} else if s.Trigger.TriggerType == TriggerTypeTimer {
		cronManager.Remove(s.cronId)
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
