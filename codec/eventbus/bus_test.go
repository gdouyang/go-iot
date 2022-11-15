package eventbus_test

import (
	"fmt"
	"go-iot/codec/eventbus"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBus(t *testing.T) {
	eventbus.Subscribe("/device/*/event/*", func(data interface{}) {
		fmt.Println("subscribe", data)
		assert.True(t, data == "11")
	})

	eventbus.Publish("/device/121/event/12", "11")
	eventbus.Publish("/device/aa/event/bb", "11")
	eventbus.Publish("/device/cc/event/dd", "11")
}
