package ruleengine

import (
	"errors"
	"fmt"
	"go-iot/pkg/core"
	"go-iot/pkg/core/tsl"
	"go-iot/pkg/eventbus"
	"sync"

	logs "go-iot/pkg/logger"

	"github.com/dop251/goja"
)

type AlarmEvent struct {
	ProductId string
	DeviceId  string
	RuleId    int64
	AlarmName string
	CreateId  int64
	Data      map[string]interface{}
}

func (e *AlarmEvent) Type() eventbus.MessageType {
	return eventbus.ALARM
}

func (e *AlarmEvent) GetDeviceId() string {
	return e.DeviceId
}

func (e *AlarmEvent) GetProductId() string {
	return e.DeviceId
}

type TriggerType string

const (
	TriggerTypeDevice TriggerType = "device"
	TriggerTypeTimer  TriggerType = "timer"
	TypeAlarm                     = "alarm"
	TypeScene                     = "scene"
	// 动作执行类型

	ActionExecutorNotifier = "notifier"
	ActionExecutorDevice   = "device-message-sender"
	// FilterType

	FilterTypeOnline     = "online"
	FilterTypeOffline    = "offline"
	FilterTypeProperties = "properties"
	FilterTypeEvent      = "event"
	// 比较运算符

	OperatorEq  = "eq"  // 等于
	OperatorNeq = "neq" // 不等于
	OperatorGt  = "gt"  // 大于(>)
	OperatorLt  = "lt"  // 小于
	OperatorGte = "gte" // 大于等于
	OperatorLte = "lte" // 小于等于

	// 事件、功能本身（事件可以是本身触发，也可以是事件的子属性触发）
	This = "this"
)

type Trigger struct {
	FilterType string            `json:"filterType"` // 触发消息类型 online,offline,properties,event
	Filters    []ConditionFilter `json:"filters"`    // 条件
	ShakeLimit ShakeLimit        `json:"shakeLimit"` // 防抖限制
	pool       *core.VmPool      `json:"-"`
}

func (t *Trigger) GetTopic(productId string) string {
	if t.FilterType == FilterTypeProperties {
		return eventbus.GetMesssageTopic(productId, "*")
	} else if t.FilterType == FilterTypeOnline {
		return eventbus.GetOnlineTopic(productId, "*")
	} else if t.FilterType == FilterTypeOffline {
		return eventbus.GetOfflineTopic(productId, "*")
	} else if t.FilterType == FilterTypeEvent {
		return eventbus.GetEventTopic(productId, "*")
	}
	logs.Errorf("filterType[%s] is illegal, must be [properties, online, offline, event]", t.FilterType)
	return ""
}

func (c *Trigger) GetExpression() string {
	var expression string
	isOr := false
	for index, v := range c.Filters {
		if index == 0 {
			expression = v.getExpression()
		}
		if index > 0 {
			if v.Logic == "and" {
				expression = fmt.Sprintf("%s && %s", expression, v.getExpression())
			} else {
				if isOr {
					expression = fmt.Sprintf("%s)", expression)
				}
				isOr = true
				expression = fmt.Sprintf("%s || (%s", expression, v.getExpression())
			}
		}
	}
	if isOr {
		expression = fmt.Sprintf("%s)", expression)
	}
	return expression
}

func (c *Trigger) Evaluate(data map[string]interface{}) (bool, error) {
	if c.pool == nil {
		var mutex sync.Mutex
		mutex.Lock()
		defer mutex.Unlock()
		pool, err := core.NewVmPool("function test() { return "+c.GetExpression()+";}", 5)
		if err != nil {
			return false, err
		}
		c.pool = pool
	}
	vm := c.pool.Get()
	defer func() {
		c.pool.Put(vm)
	}()
	fn, succ := goja.AssertFunction(vm.Get("test"))
	if !succ {
		return false, errors.New("test not a function")
	}
	result, err := fn(vm.ToValue(data))
	if err != nil {
		return false, err
	}
	val := result.ToBoolean()
	return val, nil
}

type ConditionFilter struct {
	Key      string `json:"key"`
	Value    string `json:"value"`
	Operator string `json:"operator"`        // eq, neq, gt, lt, gte, lte
	Logic    string `json:"logic,omitempty"` // and, or
	DataType string `json:"dataType"`
}

func (c *ConditionFilter) getExpression() string {
	var oper string
	switch c.Operator {
	case OperatorEq:
		oper = "=="
	case OperatorNeq:
		oper = "!="
	case OperatorGt:
		oper = ">"
	case OperatorLt:
		oper = "<"
	case OperatorGte:
		oper = ">="
	case OperatorLte:
		oper = "<="
	default:
		oper = "=="
	}
	stringTypeFunc := func(c *ConditionFilter, oper string) string {
		if oper == "==" || oper == "!=" {
			oper = oper + "="
		}
		return fmt.Sprintf("this.%s %s \"%s\"", c.Key, oper, c.Value)
	}
	switch c.DataType {
	case tsl.TypeString:
		return stringTypeFunc(c, oper)
	case tsl.TypeEnum:
		return stringTypeFunc(c, oper)
	case tsl.TypeDate:
		return stringTypeFunc(c, oper)
	case tsl.TypeBool:
		return stringTypeFunc(c, oper)
	case tsl.TypePassword:
		return stringTypeFunc(c, oper)
	case This:
		return "true" // event self is happen
	default:
		return fmt.Sprintf("this.%s %s %s", c.Key, oper, c.Value)
	}
}

// 抖动限制
type ShakeLimit struct {
	Enabled    bool  `json:"enabled"`
	Time       int32 `json:"time"`
	Threshold  int32 `json:"threshold"`
	AlarmFirst bool  `json:"alarmFirst"`
}

// 执行
type Action struct {
	Executor      string `json:"executor"`      // 执行器
	Configuration string `json:"configuration"` // 执行器配置
}
