package mqttclient

import (
	"encoding/hex"
	"fmt"
	"go-iot/codec"
	"strings"

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

type clientSession struct {
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
	codec     codec.Codec
}

func newClientSession(deviceId string, network codec.NetworkConf, spec *MQTTClientSpec) (*clientSession, error) {
	opts := MQTT.NewClientOptions()
	opts.AddBroker("tcp://" + spec.Host + ":" + fmt.Sprint(spec.Port))
	opts.SetClientID(spec.ClientId)
	opts.SetUsername(spec.Username)
	opts.SetPassword(spec.Password)
	opts.SetCleanSession(spec.CleanSession)

	session := &clientSession{
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
		logs.Info("connection lost clientId:%s, err:%s ", opts.ClientID, err.Error())
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
	session.codec = codec.GetCodec(network.ProductId)
	session.deviceOnline(deviceId)

	session.codec.OnConnect(&mqttClientContext{
		BaseContext: codec.BaseContext{
			DeviceId:  session.GetDeviceId(),
			ProductId: network.ProductId,
			Session:   session,
		},
	})

	return session, nil
}

func (s *clientSession) Publish(topic string, msg string) error {
	s.client.Publish(topic, QoS0, false, msg)
	return nil
}

func (s *clientSession) PublishHex(topic string, payload string) {
	b, err := hex.DecodeString(payload)
	if err != nil {
		logs.Error("mqtt client hex decode error:", err)
		return
	}
	s.client.Publish(topic, QoS0, false, b)
}

func (s *clientSession) PublishQos1(topic string, msg interface{}) error {
	s.client.Publish(topic, QoS1, false, msg)
	return nil
}

func (s *clientSession) Disconnect() error {
	if s.isClose {
		return nil
	}
	s.isClose = true
	s.client.Disconnect(250)
	codec.DelSession(s.deviceId)
	return nil
}

func (s *clientSession) SetDeviceId(deviceId string) {
	s.deviceId = deviceId
}

func (s *clientSession) GetDeviceId() string {
	return s.deviceId
}

func (s *clientSession) deviceOnline(deviceId string) {
	deviceId = strings.TrimSpace(deviceId)
	if len(deviceId) > 0 {
		codec.PutSession(deviceId, s)
	}
}

func (s *clientSession) readLoop() {
	defer s.Disconnect()
	for {
		select {
		case msg := <-s.choke:
			s.codec.OnMessage(&mqttClientContext{
				BaseContext: codec.BaseContext{
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
