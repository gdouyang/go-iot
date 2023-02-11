package ruleengine

import (
	"fmt"
	"go-iot/pkg/codec/eventbus"
	"sync"

	"github.com/beego/beego/v2/core/logs"
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
		logs.Error("triggerType not found")
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

func (s *RuleExecutor) subscribeEvent(data eventbus.Message) {
	if s.Trigger.FilterType == "online" || s.Trigger.FilterType == "offline" {
		s.runAction(nil)
		return
	}
	s.evaluate(data)
}

func (s *RuleExecutor) evaluate(data eventbus.Message) {
	pass := true
	var data1 map[string]interface{}
	var deviceId string
	if p, ok := data.(*eventbus.PropertiesMessage); ok {
		data1 = p.Data
		deviceId = p.DeviceId
	}
	if p, ok := data.(*eventbus.EventMessage); ok {
		data1 = p.Data
		deviceId = p.DeviceId
	}
	if len(s.deviceIdMap) > 0 {
		pass = false
		if _, ok := s.deviceIdMap[deviceId]; ok {
			pass = true
		}
	}
	if pass {
		// product := codec.GetProductManager().Get(s.ProductId)
		// if nil != product {
		// 	tsl.ValueConvert1(product.GetTslProperty(), &data1)
		// }
		pass, err := s.Trigger.Evaluate(data1)
		if err != nil {
			logs.Error(err)
			return
		}
		if pass {
			if s.Type == TypeAlarm {
				event := AlarmEvent{
					ProductId: s.ProductId,
					DeviceId:  deviceId,
					RuleId:    s.Id,
					AlarmName: s.Name,
					Data:      data1,
				}
				eventbus.Publish(eventbus.GetAlarmTopic(s.ProductId, deviceId), &event)
			}
			s.runAction(data1)
		}
	}
}

func (s *RuleExecutor) cronRun() {
	s.runAction(nil)
}

func (s *RuleExecutor) runAction(data map[string]interface{}) {
	for _, action := range s.Actions {
		if action.Executor == "device-message-sender" {
			a, err := NewDeviceCmdAction(action.Configuration)
			if err != nil {
				logs.Error(err)
			} else {
				a.Do()
			}
		} else if action.Executor == "notifier" {
			a, err := NewNotifierAction(action.Configuration, data)
			if err != nil {
				logs.Error(err)
			} else {
				a.Do()
			}
		} else {
			logs.Info("%s %s %s action is run", s.Name, s.Type, s.TriggerType)
		}
	}
}
