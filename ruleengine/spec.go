package ruleengine

import (
	"fmt"
	"go-iot/codec/eventbus"

	"gopkg.in/Knetic/govaluate.v2"
)

type TaskExecutorProvider interface {
	GetExecutor() string
}

type TriggerType string

const (
	TriggerTypeDevice TriggerType = "device"
	TriggerTypeTimer  TriggerType = "timer"
)

type Trigger struct {
	FilterType string            `json:"filterType"` // 触发消息类型 online,offline,properties,event
	Filters    []ConditionFilter `json:"filters"`    // 条件
	ShakeLimit ShakeLimit        `json:"shakeLimit"` // 防抖限制
}

func (t *Trigger) GetTopic(productId, deviceId string) string {
	if t.FilterType == "properties" {
		return eventbus.GetMesssageTopic(productId, deviceId)
	} else if t.FilterType == "online" {
		return eventbus.GetOnlineTopic(productId, deviceId)
	} else if t.FilterType == "offline" {
		return eventbus.GetOfflineTopic(productId, deviceId)
	}
	return ""
}

type ConditionFilter struct {
	Key        string                         `json:"key"`
	Value      string                         `json:"value"`
	Operator   string                         `json:"operator"`
	Logic      string                         `json:"logic,omitempty"`
	expression *govaluate.EvaluableExpression `json:"-"`
}

func (c *ConditionFilter) getExpression() string {
	var oper string
	switch c.Operator {
	case "eq":
		oper = "="
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
	}
	return fmt.Sprintf("%s %s %s", c.Key, oper, c.Value)
}

func (c *ConditionFilter) evaluate(data map[string]interface{}) (bool, error) {
	if c.expression == nil {
		expression, err := govaluate.NewEvaluableExpression(c.getExpression())
		if err != nil {
			return false, err
		}
		c.expression = expression
	}
	result, err := c.expression.Evaluate(data)
	if err != nil {
		return false, err
	}
	return result.(bool), nil
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
	Executor      string                 `json:"executor"`      // 执行器
	Configuration map[string]interface{} `json:"configuration"` // 执行器配置
}
