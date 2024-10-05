package mqttclient

import (
	"encoding/hex"
	"fmt"
	"go-iot/pkg/core"
	"go-iot/pkg/network"
	"strings"

	logs "go-iot/pkg/logger"

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

type MqttClientSession struct {
	client    MQTT.Client
	deviceId  string
	productId string
	Topics    map[string]int
	ClientID  string
	Username  string
	CleanFlag bool
	choke     chan MQTT.Message
	done      chan struct{}
	isClose   bool
	core      core.Codec
}

func newClientSession(deviceId string, network network.NetworkConf, spec *MQTTClientSpec) (*MqttClientSession, error) {
	opts := MQTT.NewClientOptions()
	opts.AddBroker("tcp://" + spec.Host + ":" + fmt.Sprint(spec.Port))
	opts.SetClientID(spec.ClientId)
	opts.SetUsername(spec.Username)
	opts.SetPassword(spec.Password)
	opts.SetCleanSession(spec.CleanSession)

	session := &MqttClientSession{
		ClientID:  spec.ClientId,
		Username:  spec.Username,
		CleanFlag: spec.CleanSession,
		Topics:    spec.Topics,
		deviceId:  deviceId,
		productId: network.ProductId,
		choke:     make(chan MQTT.Message),
		done:      make(chan struct{}),
	}
	if len(spec.Topics) == 0 {
		opts.SetDefaultPublishHandler(func(client MQTT.Client, msg MQTT.Message) {
			session.choke <- msg
		})
	}
	opts.SetConnectionLostHandler(func(c MQTT.Client, err error) {
		logs.Infof("connection lost clientId:%s, err:%s ", opts.ClientID, err.Error())
		close(session.done)
	})

	client := MQTT.NewClient(opts)
	if len(spec.Topics) > 0 {
		var filters map[string]byte = map[string]byte{}
		for key, v := range spec.Topics {
			filters[key] = byte(v)
		}
		client.SubscribeMultiple(filters, func(client MQTT.Client, msg MQTT.Message) {
			session.choke <- msg
		})
	}
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return nil, token.Error()
	}
	session.client = client
	session.core = core.GetCodec(network.ProductId)
	session.deviceOnline(deviceId)

	session.core.OnConnect(&mqttClientContext{
		BaseContext: core.BaseContext{
			DeviceId:  session.GetDeviceId(),
			ProductId: network.ProductId,
			Session:   session,
		},
	})

	return session, nil
}

func (s *MqttClientSession) Publish(topic string, msg string) error {
	s.client.Publish(topic, QoS0, false, msg)
	return nil
}

func (s *MqttClientSession) PublishHex(topic string, payload string) {
	b, err := hex.DecodeString(payload)
	if err != nil {
		logs.Errorf("mqtt client hex decode error: %v", err)
		return
	}
	s.client.Publish(topic, QoS0, false, b)
}

func (s *MqttClientSession) PublishQos1(topic string, msg interface{}) error {
	s.client.Publish(topic, QoS1, false, msg)
	return nil
}

func (s *MqttClientSession) Disconnect() error {
	if s.isClose {
		return nil
	}
	s.isClose = true
	s.client.Disconnect(250)
	core.DelSession(s.deviceId)
	return nil
}

func (s *MqttClientSession) Close() error {
	return s.Disconnect()
}

func (s *MqttClientSession) SetDeviceId(deviceId string) {
	s.deviceId = deviceId
}

func (s *MqttClientSession) GetDeviceId() string {
	return s.deviceId
}
func (s *MqttClientSession) GetInfo() map[string]any {
	return map[string]any{}
}

func (s *MqttClientSession) deviceOnline(deviceId string) {
	deviceId = strings.TrimSpace(deviceId)
	if len(deviceId) > 0 {
		core.PutSession(deviceId, s, false)
	}
}

func (s *MqttClientSession) readLoop() {
	defer s.Disconnect()
	for {
		select {
		case msg := <-s.choke:
			s.core.OnMessage(&mqttClientContext{
				BaseContext: core.BaseContext{
					DeviceId:  s.GetDeviceId(),
					ProductId: s.productId,
					Session:   s,
				},
				Data: msg,
			})
		case <-s.done:
			return
		}

	}
}
