package ruleengine_test

import (
	"encoding/json"
	"go-iot/pkg/codec"
	"go-iot/pkg/codec/eventbus"
	"go-iot/pkg/codec/tsl"
	"go-iot/pkg/ruleengine"
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
	err := ruleengine.Start(1, &rule)
	assert.Nil(t, err)
	tslData := tsl.NewTslData()
	tslData.Properties = []tsl.TslProperty{
		{Id: "light", Name: "亮度", ValueType: map[string]interface{}{"type": "int"}},
		{Id: "current", Name: "电流", ValueType: map[string]interface{}{"type": "double"}},
		{Id: "obj", Name: "obj", ValueType: map[string]interface{}{"properties": []tsl.TslProperty{{Id: "name", Name: "name", ValueType: map[string]interface{}{"type": "string"}}}}},
	}
	b, err := json.Marshal(tslData)
	assert.Nil(t, err)

	codec.DefaultManagerId = "mem"
	prod, err := codec.NewProduct("test123", map[string]string{}, codec.TIME_SERISE_MOCK, string(b))
	assert.Nil(t, err)
	assert.NotNil(t, prod)
	if prod != nil {
		codec.PutProduct(prod)
		codec.PutDevice(&codec.Device{Id: "1234"})
		eventbus.Publish(eventbus.GetMesssageTopic("test123", "1234"), &ruleengine.AlarmEvent{
			DeviceId:  "1234",
			ProductId: "test123",
			Data: map[string]interface{}{
				"deviceId": "1234",
				"light":    "32",
				"current":  "22",
				"obj":      map[string]string{"name": "test"},
			},
		})
		time.Sleep(time.Second * 1)
	}
}