package ruleengine_test

import (
	"encoding/json"
	"fmt"
	"go-iot/pkg/core"
	"go-iot/pkg/eventbus"
	"go-iot/pkg/logger"
	"go-iot/pkg/ruleengine"
	"go-iot/pkg/store"
	"go-iot/pkg/tsl"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRule(t *testing.T) {
	logger.InitNop()
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
		tsl.NewPropertyInt("light", "亮度"),
		tsl.NewPropertyFloat("current", "电流"),
		tsl.NewPropertyObject("obj", "obj", []tsl.Property{
			tsl.NewPropertyString("name", "name"),
		}),
	}
	b, err := json.Marshal(tslData)
	assert.Nil(t, err)
	fmt.Println(string(b))

	core.RegDeviceStore(store.NewMockDeviceStore())
	prod, err := core.NewProduct("test123", map[string]string{}, core.TIME_SERISE_MOCK, string(b))
	assert.Nil(t, err)
	assert.NotNil(t, prod)
	core.PutProduct(prod)
	core.PutDevice(&core.Device{Id: "1234"})
	eventbus.PublishProperties(&eventbus.PropertiesMessage{
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
