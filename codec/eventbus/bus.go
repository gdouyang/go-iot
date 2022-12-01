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
		m:       map[string][]func(data Message){},
		matcher: *NewAntPathMatcher(),
	}
}

type eventBus struct {
	sync.Mutex
	m       map[string][]func(data Message)
	matcher AntPathMatcher
}

func (b *eventBus) match(pattern string, path string) bool {
	return bus.matcher.Match(pattern, path)
}

func Subscribe(pattern string, run func(msg Message)) {
	bus.Lock()
	defer bus.Unlock()
	bus.m[pattern] = append(bus.m[pattern], run)
}

func UnSubscribe(pattern string, run func(data Message)) {
	bus.Lock()
	defer bus.Unlock()
	listener := bus.m[pattern]
	var l1 []func(data Message)
	for _, callback := range listener {
		sf1 := reflect.ValueOf(callback)
		sf2 := reflect.ValueOf(run)
		if sf1.Pointer() != sf2.Pointer() {
			l1 = append(l1, callback)
		}
	}
	bus.m[pattern] = l1
}

func Publish(topic string, data Message) {
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

// publish event of tsl properties
func PublishProperties(data *PropertiesMessage) {
	Publish(GetMesssageTopic(data.ProductId, data.DeviceId), data)
}

// publish event of tsl events
func PublishEvent(data *EventMessage) {
	Publish(GetEventTopic(data.ProductId, data.DeviceId), data)
}

type MessageType string

const (
	PROP    MessageType = "prop"
	EVENT   MessageType = "event"
	ALARM   MessageType = "alarm"
	ONLINE  MessageType = "online"
	OFFLINE MessageType = "offline"
)

type Message interface {
	Type() MessageType
}

type PropertiesMessage struct {
	DeviceId  string
	ProductId string
	Data      map[string]interface{}
}

func NewPropertiesMessage(deviceId string, productId string, data map[string]interface{}) PropertiesMessage {
	return PropertiesMessage{
		DeviceId:  deviceId,
		ProductId: productId,
		Data:      data,
	}
}

func (m *PropertiesMessage) Type() MessageType {
	return PROP
}

type EventMessage struct {
	DeviceId  string
	ProductId string
	Data      map[string]interface{}
}

func NewEventMessage(deviceId string, productId string, data map[string]interface{}) EventMessage {
	return EventMessage{
		DeviceId:  deviceId,
		ProductId: productId,
		Data:      data,
	}
}

func (m *EventMessage) Type() MessageType {
	return EVENT
}

type OnlineMessage struct {
	DeviceId  string
	ProductId string
}

func (m *OnlineMessage) Type() MessageType {
	return ONLINE
}

type OfflineMessage struct {
	DeviceId  string
	ProductId string
}

func (m *OfflineMessage) Type() MessageType {
	return OFFLINE
}
