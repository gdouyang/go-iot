package eventbus

import (
	"sync"
)

const (
	// /device/productId/deviceId/property
	DeviceMessageTopic string = "/device/%s/%s/property"
)

var b = &eventBus{
	m: map[string][]func(data interface{}){},
}

type eventBus struct {
	sync.Mutex
	m map[string][]func(data interface{})
}

func Subscribe(topic string, run func(data interface{})) {
	b.Lock()
	defer b.Unlock()
	if _, ok := b.m[topic]; ok {
		b.m[topic] = append(b.m[topic], run)
	}
}

func Publish(topic string, data interface{}) {
	b.Lock()
	defer b.Unlock()
	if _, ok := b.m[topic]; ok {
		listener := b.m[topic]
		for _, callback := range listener {
			callback(data)
		}
	}
}
