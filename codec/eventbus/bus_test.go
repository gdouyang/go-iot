package eventbus_test

import (
	"fmt"
	"go-iot/codec/eventbus"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBus(t *testing.T) {
	eventbus.Subscribe("/device/*/event/*", func(data eventbus.Message) {
		fmt.Println("subscribe", data)
		assert.True(t, data.Type() == eventbus.PROP)
		if p, ok := data.(*eventbus.PropertiesMessage); ok {
			assert.True(t, p.Data["test"] == "11")
		}
	})

	event := eventbus.NewPropertiesMessage("1234", "test123", map[string]interface{}{"test": "11"})

	eventbus.Publish("/device/121/event/12", &event)
	eventbus.Publish("/device/aa/event/bb", &event)
	eventbus.Publish("/device/cc/event/dd", &event)
}
