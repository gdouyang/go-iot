package ruleengine_test

import (
	"go-iot/codec"
	"go-iot/codec/eventbus"
	"go-iot/codec/tsl"
	"go-iot/ruleengine"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRule(t *testing.T) {
	trigger := ruleengine.Trigger{
		FilterType: "properties",
		Filters: []ruleengine.ConditionFilter{
			{Key: "light", Operator: "eq", Value: "321"},
			{Logic: "or", Key: "current", Operator: "eq", Value: "22"},
			{Logic: "and", Key: "obj.name", Operator: "eq", Value: "test", DataType: "string"},
		},
	}
	var rule = ruleengine.RuleExecutor{
		Name:        "test",
		Type:        "scene",
		TriggerType: ruleengine.TriggerTypeDevice,
		ProductId:   "test123",
		DeviceIds:   []string{"1234"},
		Trigger:     trigger,
		Actions:     []ruleengine.Action{{Executor: "console"}},
	}
	err := ruleengine.StartScene(1, &rule)
	assert.Nil(t, err)
	var propMap = map[string]tsl.TslProperty{
		"light":   {Id: "light", Name: "亮度", ValueType: map[string]interface{}{"type": "int"}},
		"current": {Id: "current", Name: "电流", ValueType: map[string]interface{}{"type": "double"}},
		"obj":     {Id: "obj", Name: "obj", ValueType: map[string]interface{}{"properties": []tsl.TslProperty{{Id: "name", Name: "name", ValueType: map[string]interface{}{"type": "string"}}}}},
	}
	codec.DefaultManagerId = "mem"
	codec.GetProductManager().Put(&codec.DefaultProdeuct{Id: "test123", TslProperty: propMap})
	codec.GetDeviceManager().Put(&codec.DefaultDevice{Id: "1234"})
	eventbus.Publish(eventbus.GetMesssageTopic("test123", "1234"), map[string]interface{}{
		"deviceId": "1234",
		"light":    "32",
		"current":  "22",
		"obj":      map[string]string{"name": "test"},
	})
	time.Sleep(time.Second * 1)
}
