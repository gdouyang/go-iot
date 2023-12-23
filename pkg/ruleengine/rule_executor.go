package ruleengine

import (
	"errors"
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

// 校验
func (s *RuleExecutor) Valid() error {
	if s.TriggerType != TriggerTypeDevice && s.TriggerType != TriggerTypeTimer {
		return errors.New("triggerType must be [device, timer]")
	}
	if s.Type != TypeAlarm && s.Type != TypeScene {
		return errors.New("type must be [scene, alarm]")
	}
	if s.TriggerType == TriggerTypeDevice && len(s.ProductId) == 0 {
		return errors.New("productId must persent")
	}
	if s.TriggerType == TriggerTypeTimer && len(s.Cron) == 0 {
		return errors.New("cron must persent")
	}
	if s.TriggerType == TriggerTypeDevice {
		if s.Trigger.FilterType != FilterTypeOnline &&
			s.Trigger.FilterType != FilterTypeOffline &&
			s.Trigger.FilterType != FilterTypeProperties &&
			s.Trigger.FilterType != FilterTypeEvent {
			return errors.New("trigger.filterType must be [online, offline, properties, event]")
		}
		if s.Trigger.FilterType == FilterTypeProperties ||
			s.Trigger.FilterType == FilterTypeEvent {
			if len(s.Trigger.Filters) == 0 {
				return errors.New("trigger.filters must not empty")
			}
			for i, v := range s.Trigger.Filters {
				if i > 0 && v.Logic != "and" && v.Logic != "or" {
					return fmt.Errorf("trigger.filters[%d].logic must be [and, or]", i)
				}
				if len(v.Key) == 0 {
					return fmt.Errorf("trigger.filters[%d].key must persent", i)
				}
				// 事件本身(可以不填写值)
				if v.DataType == This {
					continue
				}
				if len(v.Operator) == 0 {
					return fmt.Errorf("trigger.filters[%d].operator must persent", i)
				}
				if v.Operator != OperatorEq && v.Operator != OperatorNeq &&
					v.Operator != OperatorGt && v.Operator != OperatorLt &&
					v.Operator != OperatorGte && v.Operator != OperatorLte {
					return fmt.Errorf("trigger.filters[%d].operator must be [eq, neq, gt, lt, gte, lte]", i)
				}
				if len(v.Value) == 0 {
					return fmt.Errorf("trigger.filters[%d].value must persent", i)
				}
			}
		}
	}
	if len(s.Actions) == 0 {
		return errors.New("actions must not empty")
	}
	for _, v := range s.Actions {
		if v.Executor != ActionExecutorDevice && v.Executor != ActionExecutorNotifier {
			return errors.New("action executor must be [notifier, device-message-sender]")
		}
	}
	return nil
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
			return fmt.Errorf("cron表达式错误: %s", err.Error())
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
		} else {
			logs.Infof("unsupported msg type: %s", msg.Type())
			return
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
			data["deviceId"] = deviceId
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
		} else if action.Executor == "console" {
			logs.Infof("exec name: [%s], type: [%s], triggerType: [%s], executor: [%s], data: %v", s.Name, s.Type, s.TriggerType, action.Executor, data)
		} else {
			logs.Warnf("unsupported executor [%s], name: [%s], type: [%s], triggerType: [%s], ", action.Executor, s.Name, s.Type, s.TriggerType)
		}
	}
}
