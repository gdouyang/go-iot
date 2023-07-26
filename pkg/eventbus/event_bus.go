package eventbus

import (
	"fmt"
	"time"
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

type MessageType string

const (
	PROP      MessageType = "property"
	EVENT     MessageType = "event"
	ALARM     MessageType = "alarm"
	ONLINE    MessageType = "online"
	OFFLINE   MessageType = "offline"
	timeformt             = "2006-01-02 15:04:05.000"
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

func newEventBus() *SingleNodeEventBus {
	return &SingleNodeEventBus{
		m:       map[string][]func(data Message){},
		matcher: *NewAntPathMatcher(),
	}
}

func Subscribe(pattern string, run func(msg Message)) {
	bus.sub(pattern, run)
}

func UnSubscribe(pattern string, run func(data Message)) {
	bus.unsub(pattern, run)
}

func Publish(topic string, data Message) {
	bus.publish(topic, data)
}

// publish event of tsl properties
func PublishProperties(data *PropertiesMessage) {
	Publish(GetMesssageTopic(data.ProductId, data.DeviceId), data)
}

// publish event of tsl events
func PublishEvent(data *EventMessage) {
	Publish(GetEventTopic(data.ProductId, data.DeviceId), data)
}
func PublishOnline(data *OnlineMessage) {
	Publish(GetOnlineTopic(data.ProductId, data.DeviceId), data)
}
func PublishOffline(data *OfflineMessage) {
	Publish(GetOfflineTopic(data.ProductId, data.DeviceId), data)
}

type Message interface {
	Type() MessageType
	GetDeviceId() string
	GetProductId() string
}

// PropertiesMessage
type PropertiesMessage struct {
	Typ        string                 `json:"type"`
	DeviceId   string                 `json:"deviceId"`
	ProductId  string                 `json:"productId"`
	CreateTime string                 `json:"createTime"`
	Data       map[string]interface{} `json:"data"`
}

func NewPropertiesMessage(deviceId string, productId string, data map[string]interface{}) PropertiesMessage {
	return PropertiesMessage{
		Typ:        string(PROP),
		DeviceId:   deviceId,
		ProductId:  productId,
		CreateTime: time.Now().Format(timeformt),
		Data:       data,
	}
}

func (m *PropertiesMessage) Type() MessageType {
	return PROP
}
func (m *PropertiesMessage) GetDeviceId() string {
	return m.DeviceId
}
func (m *PropertiesMessage) GetProductId() string {
	return m.ProductId
}

// EventMessage
type EventMessage struct {
	Typ        string                 `json:"type"`
	DeviceId   string                 `json:"deviceId"`
	ProductId  string                 `json:"productId"`
	EventId    string                 `json:"eventId"`
	CreateTime string                 `json:"createTime"`
	Data       map[string]interface{} `json:"data"`
}

func NewEventMessage(deviceId string, productId string, eventId string, data map[string]interface{}) EventMessage {
	return EventMessage{
		Typ:        string(EVENT),
		DeviceId:   deviceId,
		ProductId:  productId,
		EventId:    eventId,
		Data:       data,
		CreateTime: time.Now().Format(timeformt),
	}
}

func (m *EventMessage) Type() MessageType {
	return EVENT
}

func (m *EventMessage) GetDeviceId() string {
	return m.DeviceId
}
func (m *EventMessage) GetProductId() string {
	return m.ProductId
}

// OnlineMessage
type OnlineMessage struct {
	Typ       string `json:"type"`
	DeviceId  string `json:"deviceId"`
	ProductId string `json:"productId"`
}

func NewOnlineMessage(deviceId string, productId string) OnlineMessage {
	return OnlineMessage{
		Typ:       string(ONLINE),
		DeviceId:  deviceId,
		ProductId: productId,
	}
}

func (m *OnlineMessage) Type() MessageType {
	return ONLINE
}

func (m *OnlineMessage) GetDeviceId() string {
	return m.DeviceId
}

func (m *OnlineMessage) GetProductId() string {
	return m.ProductId
}

// OfflineMessage
type OfflineMessage struct {
	Typ       string `json:"type"`
	DeviceId  string `json:"deviceId"`
	ProductId string `json:"productId"`
}

func NewOfflineMessage(deviceId string, productId string) OfflineMessage {
	return OfflineMessage{
		Typ:       string(OFFLINE),
		DeviceId:  deviceId,
		ProductId: productId,
	}
}

func (m *OfflineMessage) Type() MessageType {
	return OFFLINE
}
func (m *OfflineMessage) GetDeviceId() string {
	return m.DeviceId
}
func (m *OfflineMessage) GetProductId() string {
	return m.ProductId
}
