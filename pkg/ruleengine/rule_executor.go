package ruleengine

import (
	"fmt"
	"go-iot/pkg/core"
	"go-iot/pkg/eventbus"
	"sync"

	logs "go-iot/pkg/logger"

	"github.com/robfig/cron/v3"
)

var manager = &sceneManager{
	m: map[int64]*RuleExecutor{},
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
	m map[int64]*RuleExecutor
}

func Start(id int64, rule *RuleExecutor) error {
	manager.Lock()
	defer manager.Unlock()
	if len(rule.Type) == 0 {
		return fmt.Errorf("type must persent [scene,alarm]")
	}
	if len(rule.TriggerType) == 0 {
		return fmt.Errorf("triggerType must persent [device, timer]")
	}
	rule.deviceIdsToMap()
	err := rule.start()
	if err != nil {
		return err
	}
	rule.Id = id
	manager.m[id] = rule
	return nil
}

func Stop(id int64) {
	manager.Lock()
	defer manager.Unlock()
	if e, ok := manager.m[id]; ok {
		e.stop()
	}
}

type RuleExecutor struct {
	Id          int64
	Name        string
	Type        string      // scene,alarm
	TriggerType TriggerType // device, timer
	Cron        string
	ProductId   string
	DeviceIds   []string
	Trigger     Trigger
	Actions     []Action
	cronId      cron.EntryID
	deviceIdMap map[string]bool
}

func (s *RuleExecutor) deviceIdsToMap() {
	if s.deviceIdMap == nil {
		s.deviceIdMap = map[string]bool{}
	}
	for _, v := range s.DeviceIds {
		s.deviceIdMap[v] = true
	}
}

func (s *RuleExecutor) start() error {
	if s.TriggerType == TriggerTypeDevice {
		topic := s.Trigger.GetTopic(s.ProductId)
		eventbus.Subscribe(topic, s.subscribeEvent)
		return nil
	} else if s.TriggerType == TriggerTypeTimer {
		entryID, err := cronManager.AddFunc(s.Cron, s.cronRun)
		if err != nil {
			return err
		}
		s.cronId = entryID
	} else {
		logs.Errorf("triggerType not found [%s]", s.TriggerType)
	}
	return nil
}

func (s *RuleExecutor) stop() {
	if s.TriggerType == TriggerTypeDevice {
		eventbus.UnSubscribe(s.Trigger.GetTopic(s.ProductId), s.subscribeEvent)
	} else if s.TriggerType == TriggerTypeTimer {
		cronManager.Remove(s.cronId)
	}
}

// 事件触发
func (s *RuleExecutor) subscribeEvent(msg eventbus.Message) {
	s.evaluate(msg)
}

func (s *RuleExecutor) evaluate(msg eventbus.Message) {
	pass := true
	var data map[string]interface{}
	deviceId := msg.GetDeviceId()
	if s.Trigger.FilterType == core.ONLINE || s.Trigger.FilterType == core.OFFLINE {
		data = map[string]interface{}{"deviceId": deviceId, "state": s.Trigger.FilterType}
	} else {
		if p, ok := msg.(*eventbus.PropertiesMessage); ok {
			data = p.Data
		} else if p, ok := msg.(*eventbus.EventMessage); ok {
			data = p.Data
		}
		// 指定了设备
		if len(s.deviceIdMap) > 0 {
			pass = false
			if _, ok := s.deviceIdMap[deviceId]; ok {
				pass = true
			}
		}
		if pass {
			evalPass, err := s.Trigger.Evaluate(data)
			if err != nil {
				logs.Errorf("rule trigger evaluate error: %v", err)
				return
			}
			pass = evalPass
		} else {
			logs.Debugf("device %s skip", deviceId)
		}
	}
	if pass {
		s.createAlarm(deviceId, data)
		s.runAction(data)
	}
}

// 定时任务触发
func (s *RuleExecutor) cronRun() {
	s.runAction(nil)
}

func (s *RuleExecutor) createAlarm(deviceId string, data map[string]interface{}) {
	if s.Type == TypeAlarm {
		event := AlarmEvent{
			ProductId: s.ProductId,
			DeviceId:  deviceId,
			RuleId:    s.Id,
			AlarmName: s.Name,
			Data:      data,
		}
		eventbus.Publish(eventbus.GetAlarmTopic(s.ProductId, deviceId), &event)
	}
}

func (s *RuleExecutor) runAction(data map[string]interface{}) {
	for _, action := range s.Actions {
		if action.Executor == "device-message-sender" {
			a, err := NewDeviceCmdAction(action.Configuration)
			if err != nil {
				logs.Errorf("rule executor run action [device-message-sender] error: %v", err)
			} else {
				a.Do()
			}
		} else if action.Executor == "notifier" {
			a, err := NewNotifierAction(action.Configuration, data)
			if err != nil {
				logs.Errorf("rule executor run action [notifier] error: %v", err)
			} else {
				a.Do()
			}
		} else {
			logs.Warnf("%s %s %s %s Executor not support", s.Name, s.Type, s.TriggerType, action.Executor)
		}
	}
}
