package ruleengine

import (
	"fmt"
	"go-iot/codec/eventbus"
	"go-iot/codec/tsl"
	"sync"

	"github.com/beego/beego/v2/core/logs"
	"github.com/robertkrimen/otto"
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

type TriggerType string

const (
	TriggerTypeDevice TriggerType = "device"
	TriggerTypeTimer  TriggerType = "timer"
	TypeAlarm                     = "alarm"
	TypeScene                     = "scene"
)

type Trigger struct {
	FilterType string            `json:"filterType"` // 触发消息类型 online,offline,properties,event
	Filters    []ConditionFilter `json:"filters"`    // 条件
	ShakeLimit ShakeLimit        `json:"shakeLimit"` // 防抖限制
	expression *otto.Otto        `json:"-"`
}

func (t *Trigger) GetTopic(productId string) string {
	if t.FilterType == "properties" {
		return eventbus.GetMesssageTopic(productId, "*")
	} else if t.FilterType == "online" {
		return eventbus.GetOnlineTopic(productId, "*")
	} else if t.FilterType == "offline" {
		return eventbus.GetOfflineTopic(productId, "*")
	} else if t.FilterType == "event" {
		return eventbus.GetEventTopic(productId, "*")
	}
	logs.Error("filterType[%s] is illegal, must be [properties, online, offline, event]", t.FilterType)
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
	if c.expression == nil {
		var mutex sync.Mutex
		mutex.Lock()
		defer mutex.Unlock()
		vm := otto.New()
		_, err := vm.Run("function test() { return " + c.GetExpression() + ";}")
		// expression, err := govaluate.NewEvaluableExpression(c.GetExpression())
		if err != nil {
			return false, err
		}
		c.expression = vm
	}
	vm := c.expression.Copy()
	for key, value := range data {
		vm.Set(key, value)
	}
	result, err := vm.Call(`test`, nil)
	if err != nil {
		return false, err
	}
	val, err := result.ToBoolean()
	if err != nil {
		return false, err
	}
	return val, nil
}

type ConditionFilter struct {
	Key      string `json:"key"`
	Value    string `json:"value"`
	Operator string `json:"operator"`
	Logic    string `json:"logic,omitempty"`
	DataType string `json:"dataType"`
}

func (c *ConditionFilter) getExpression() string {
	var oper string
	switch c.Operator {
	case "eq":
		oper = "=="
	case "not":
		oper = "!="
	case "qt":
		oper = ">"
	case "lt":
		oper = "<"
	case "qte":
		oper = ">="
	case "lte":
		oper = "<="
	default:
		oper = "=="
	}
	switch c.DataType {
	case tsl.TypeString:
	case tsl.TypeEnum:
	case tsl.TypeDate:
	case tsl.TypeBool:
	case tsl.TypePassword:
		if oper == "==" || oper == "!=" {
			oper = oper + "="
		}
		return fmt.Sprintf("%s %s \"%s\"", c.Key, oper, c.Value)
	case "this":
		return "true" // event self is happen
	default:
		return fmt.Sprintf("%s %s %s", c.Key, oper, c.Value)
	}
	return ""
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
