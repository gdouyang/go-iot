package ruleengine

import (
	"go-iot/codec"
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

func StartScene(id int64, rule RuleExecutor) error {
	manager.Lock()
	defer manager.Unlock()
	e := &rule
	if e.deviceIdMap == nil {
		e.deviceIdMap = map[string]bool{}
	}
	for _, v := range e.DeviceIds {
		e.deviceIdMap[v] = true
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

type RuleExecutor struct {
	Type        string
	TriggerType TriggerType
	Cron        string
	ProductId   string
	DeviceIds   []string
	Trigger     Trigger
	Actions     []Action
	cronId      cron.EntryID
	deviceIdMap map[string]bool
}

func (s *RuleExecutor) start() error {
	if s.TriggerType == TriggerTypeDevice {
		topic := s.Trigger.GetTopic(s.ProductId, "*")
		eventbus.Subscribe(topic, s.subscribeEvent)
		return nil
	} else if s.TriggerType == TriggerTypeTimer {
		entryID, err := cronManager.AddFunc(s.Cron, s.runAction)
		if err != nil {
			return err
		}
		s.cronId = entryID
	}
	return nil
}

func (s *RuleExecutor) stop() {
	if s.TriggerType == TriggerTypeDevice {
		eventbus.UnSubscribe(eventbus.GetMesssageTopic(s.ProductId, "*"), s.subscribeEvent)
	} else if s.TriggerType == TriggerTypeTimer {
		cronManager.Remove(s.cronId)
	}
}

func (s *RuleExecutor) subscribeEvent(data interface{}) {
	pass := true
	if len(s.deviceIdMap) > 0 {
		pass = false
		m := data.(map[string]string)
		if _, ok := s.deviceIdMap[m[tsl.PropertyDeviceId]]; ok {
			pass = true
		}
	}
	if pass {
		data1 := data.(map[string]interface{})
		product := codec.GetProductManager().Get(s.ProductId)
		if nil != product {
			tsl.ValueConvert1(product.GetTslProperty(), &data1)
		}
		pass, _ = s.Trigger.Evaluate(data1)
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
		}
	}
}
