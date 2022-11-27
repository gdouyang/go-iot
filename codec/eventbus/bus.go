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
	// alarm topic pattern
	DeviceAlarmTopic string = "/device/%s/%s/alarm"
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

// /device/{productId}/{deviceId}/alarm
func GetAlarmTopic(productId string, deviceId string) string {
	return fmt.Sprintf(DeviceAlarmTopic, productId, deviceId)
}

var bus = newEventBus()

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
	return bus.matcher.Match(pattern, path)
}

func Subscribe(pattern string, run func(data interface{})) {
	bus.Lock()
	defer bus.Unlock()
	bus.m[pattern] = append(bus.m[pattern], run)
}

func UnSubscribe(pattern string, run func(data interface{})) {
	bus.Lock()
	defer bus.Unlock()
	listener := bus.m[pattern]
	var l1 []func(data interface{})
	for _, callback := range listener {
		sf1 := reflect.ValueOf(callback)
		sf2 := reflect.ValueOf(run)
		if sf1.Pointer() != sf2.Pointer() {
			l1 = append(l1, callback)
		}
	}
	bus.m[pattern] = l1
}

func Publish(topic string, data interface{}) {
	bus.Lock()
	defer bus.Unlock()
	for pattern, listener := range bus.m {
		if bus.match(pattern, topic) {
			for _, callback := range listener {
				go callback(data)
			}
		}
	}
}
