package mqttclient

import (
	"fmt"
	"go-iot/codec"

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
	productId string
	Topics    map[string]int
	ClientID  string
	Username  string
	CleanFlag bool
	choke     chan MQTT.Message
	codec     codec.Codec
}

func newClientSession(network codec.Network, spec *MQTTClientSpec) *ClientSession {
	opts := MQTT.NewClientOptions()
	opts.AddBroker("tcp://" + spec.Host + ":" + fmt.Sprint(spec.Port))
	opts.SetClientID(spec.ClientId)
	opts.SetUsername(spec.Username)
	opts.SetPassword(spec.Password)
	opts.SetCleanSession(spec.CleanSession)

	choke := make(chan MQTT.Message)
	opts.SetDefaultPublishHandler(func(client MQTT.Client, msg MQTT.Message) {
		choke <- msg
	})

	client := MQTT.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		logs.Error(token.Error())
		return nil
	}
	c := codec.NewCodec(network)
	session := &ClientSession{
		client:    client,
		ClientID:  spec.ClientId,
		Username:  spec.Username,
		CleanFlag: spec.CleanSession,
		Topics:    spec.Topics,
		productId: network.ProductId,
		choke:     choke,
		codec:     c,
	}

	c.OnConnect(&mqttClientContext{
		BaseContext: codec.BaseContext{
			DeviceId:  session.GetDeviceId(),
			ProductId: network.ProductId,
			Session:   session,
		},
	})

	return session
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

func (s *ClientSession) readLoop() {
	for {
		msg := <-s.choke
		s.codec.OnMessage(&mqttClientContext{
			BaseContext: codec.BaseContext{
				DeviceId:  s.GetDeviceId(),
				ProductId: s.productId,
				Session:   s,
			},
			Data: msg,
		})
	}
}
