package ruleengine

import (
	"fmt"

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

type SceneTrigger struct {
	TriggerType TriggerType        `json:"triggerType"`
	Cron        string             `json:"cron,omitempty"`
	ModelId     string             `json:"modelId,omitempty"` // 物模型表示,如:属性ID,事件ID
	ProductId   string             `json:"productId,omitempty"`
	DeviceId    string             `json:"deviceId,omitempty"`
	Trigger     SceneTriggerDevice `json:"trigger,omitempty"`
}

type SceneTriggerDevice struct {
	ShakeLimit ShakeLimit        `json:"shakeLimit"` // 防抖限制
	FilterType string            `json:"filterType"` // 触发消息类型
	Filters    []ConditionFilter `json:"filters"`    // 条件
}

type ConditionFilter struct {
	Key        string                         `json:"key"`
	Value      string                         `json:"value"`
	Operator   string                         `json:"operator"`
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
