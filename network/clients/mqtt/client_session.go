package mqttclient

import (
	"github.com/beego/beego/v2/core/logs"
	MQTT "github.com/eclipse/paho.mqtt.golang"
)

const (
	// Connected is MQTT client status of Connected
	Connected = 1
	// Disconnected is MQTT client status of Disconnected
	Disconnected = 2

	// QoS0 for "At most once"
	QoS0 byte = 0
	// QoS1 for "At least once
	QoS1 byte = 1
	// QoS2 for "Exactly once"
	QoS2 byte = 2
)

type ClientSession struct {
	client    MQTT.Client
	deviceId  string
	Topics    map[string]int
	ClientID  string
	Username  string
	CleanFlag bool
}

func (s *ClientSession) Send(msg interface{}) error {
	switch t := msg.(type) {
	case map[string]interface{}:
		s.client.Publish(t["topic"].(string), QoS0, false, msg.([]byte))
	default:
		logs.Error("msg must map")
	}
	return nil
}

func (s *ClientSession) Disconnect() error {
	s.client.Disconnect(250)
	return nil
}

func (s *ClientSession) SetDeviceId(deviceId string) {
	s.deviceId = deviceId
}

func (s *ClientSession) GetDeviceId() string {
	return s.deviceId
}
