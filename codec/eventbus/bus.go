package eventbus

import (
	"fmt"
	"sync"
)

const (
	// device property topic
	DeviceMessageTopic string = "/device/%s/%s/property"
	// device online
	DeviceOnlineTopic string = "/device/%s/%s/online"
	// device offline
	DeviceOfflineTopic string = "/device/%s/%s/offline"
)

// /device/{productId}/{deviceId}/property
func GetDeviceMesssageTopic(productId string, deviceId string) string {
	return fmt.Sprintf(DeviceMessageTopic, productId, deviceId)
}

var b = newEventBus()

func newEventBus() *eventBus {
	return &eventBus{
		m:       map[string][]func(data interface{}){},
		matcher: *NewAntPathMatcher(),
	}
}

type eventBus struct {
	sync.Mutex
	m       map[string][]func(data interface{})
	matcher AntPathMatcher
}

func (b *eventBus) match(pattern string, path string) bool {
	return b.matcher.Match(pattern, path)
}

func Subscribe(pattern string, run func(data interface{})) {
	b.Lock()
	defer b.Unlock()
	b.m[pattern] = append(b.m[pattern], run)
}

func Publish(topic string, data interface{}) {
	b.Lock()
	defer b.Unlock()
	for pattern, listener := range b.m {
		if b.match(pattern, topic) {
			for _, callback := range listener {
				callback(data)
			}
		}
	}
}
