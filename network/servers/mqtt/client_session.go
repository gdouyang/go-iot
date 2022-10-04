package mqttserver

import (
	"github.com/beego/beego/v2/core/logs"
	MQTT "github.com/eclipse/paho.mqtt.golang"
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
		newMsg(t["topic"].(string), msg.([]byte), QoS0)
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
