package ruleengine

import (
	"fmt"
	"go-iot/codec/eventbus"
	"go-iot/codec/tsl"
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

func StartScene(id int64, rule *RuleExecutor) error {
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
	manager.m[id] = rule
	return nil
}

func StopScene(id int64) {
	manager.Lock()
	defer manager.Unlock()
	if e, ok := manager.m[id]; ok {
		e.stop()
	}
}

type RuleExecutor struct {
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
		entryID, err := cronManager.AddFunc(s.Cron, s.runAction)
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

func (s *RuleExecutor) subscribeEvent(data interface{}) {
	pass := true
	data1 := data.(map[string]interface{})
	if len(s.deviceIdMap) > 0 {
		pass = false
		deviceId := fmt.Sprintf("%v", data1[tsl.PropertyDeviceId])
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
			s.runAction()
		}
	}
}

func (s *RuleExecutor) runAction() {
	for _, action := range s.Actions {
		if action.Executor == "device-message-sender" {
			a := DeviceCmdAction{}
			err := a.Covent(action.Configuration)
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
