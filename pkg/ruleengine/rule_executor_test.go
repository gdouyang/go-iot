package ruleengine_test

import (
	"encoding/json"
	"go-iot/pkg/core"
	"go-iot/pkg/core/store"
	"go-iot/pkg/core/tsl"
	"go-iot/pkg/eventbus"
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
	tslData.Properties = []tsl.Property{
		&tsl.PropertyInt{TslProperty: tsl.TslProperty{Id: "light", Name: "亮度"}},
		&tsl.PropertyFloat{TslProperty: tsl.TslProperty{Id: "current", Name: "电流"}},
		&tsl.PropertyObject{
			TslProperty: tsl.TslProperty{Id: "obj", Name: "obj"},
			Properties: []tsl.Property{
				&tsl.PropertyString{TslProperty: tsl.TslProperty{Id: "name", Name: "name"}},
			},
		},
	}
	b, err := json.Marshal(tslData)
	assert.Nil(t, err)

	core.RegDeviceStore(store.NewMockDeviceStore())
	prod, err := core.NewProduct("test123", map[string]string{}, core.TIME_SERISE_MOCK, string(b))
	assert.Nil(t, err)
	assert.NotNil(t, prod)
	if prod != nil {
		core.PutProduct(prod)
		core.PutDevice(&core.Device{Id: "1234"})
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
