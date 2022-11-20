package eventbus

import (
	"fmt"
	"reflect"
	"sync"
)

const (
	// device property topic pattern
	DeviceMessageTopic string = "/device/%s/%s/property"
	// device online topic pattern
	DeviceOnlineTopic string = "/device/%s/%s/online"
	// device offline topic pattern
	DeviceOfflineTopic string = "/device/%s/%s/offline"
	// event topic pattern
	DeviceEventTopic string = "/device/%s/%s/event"
)

// /device/{productId}/{deviceId}/property
func GetMesssageTopic(productId string, deviceId string) string {
	return fmt.Sprintf(DeviceMessageTopic, productId, deviceId)
}

// /device/{productId}/{deviceId}/online
func GetOnlineTopic(productId string, deviceId string) string {
	return fmt.Sprintf(DeviceOnlineTopic, productId, deviceId)
}

// /device/{productId}/{deviceId}/offline
func GetOfflineTopic(productId string, deviceId string) string {
	return fmt.Sprintf(DeviceOfflineTopic, productId, deviceId)
}

// /device/{productId}/{deviceId}/event
func GetEventTopic(productId string, deviceId string) string {
	return fmt.Sprintf(DeviceEventTopic, productId, deviceId)
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

func UnSubscribe(pattern string, run func(data interface{})) {
	b.Lock()
	defer b.Unlock()
	listener := b.m[pattern]
	var l1 []func(data interface{})
	for _, callback := range listener {
		sf1 := reflect.ValueOf(callback)
		sf2 := reflect.ValueOf(run)
		if sf1.Pointer() != sf2.Pointer() {
			l1 = append(l1, callback)
		}
	}
	b.m[pattern] = l1
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
