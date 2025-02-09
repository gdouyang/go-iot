package mqtt5

import (
	"encoding/hex"
	"go-iot/pkg/core"
	logs "go-iot/pkg/logger"
	"sync"

	mqtt "github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/packets"
)

const (
	// QoS0 for "At most once"
	QoS0 byte = 0
	// QoS1 for "At least once
	QoS1 byte = 1
	// QoS2 for "Exactly once"
	QoS2 byte = 2
)

type (
	// ClientInfo contains basic information about the MQTT5 client
	ClientInfo struct {
		cid      string
		username string
		password string
		deviceId string
		Topics   map[string]byte
	}

	// ClientAndSession represents a MQTT5 client connection
	ClientAndSession struct {
		sync.Mutex

		broker  *Broker
		client  *mqtt.Client
		info    ClientInfo
		isClose bool
		done    chan struct{}
	}

	// Message represents a message in the session
	Message struct {
		Topic   string
		payload []byte
		QoS     byte
	}
)

func NewClient(cl *mqtt.Client, broker *Broker) *ClientAndSession {
	info := ClientInfo{
		cid:      cl.ID,
		username: string(cl.Properties.Username),
	}

	client := &ClientAndSession{
		broker: broker,
		client: cl,
		info:   info,
		done:   make(chan struct{}),
	}

	return client
}

func (c *ClientAndSession) ClientID() string {
	return c.info.cid
}

func (c *ClientAndSession) UserName() string {
	return c.info.username
}

func (c *ClientAndSession) Done() <-chan struct{} {
	return c.done
}

// device session functions
func (s *ClientAndSession) Publish(topic string, payload string) {
	var qos byte
	qos, ok := s.info.Topics[topic]
	if !ok {
		qos = QoS0
	}
	msg := &Message{
		Topic:   topic,
		payload: []byte(payload),
		QoS:     byte(qos),
	}
	s.sendMessage(msg)
}

func (s *ClientAndSession) PublishHex(topic string, payload string) {
	b, err := hex.DecodeString(payload)
	if err != nil {
		logs.Errorf("mqtt hex decode error: %v", err)
		return
	}
	var qos byte
	qos, ok := s.info.Topics[topic]
	if !ok {
		qos = byte(QoS0)
	}
	msg := &Message{
		Topic:   topic,
		payload: b,
		QoS:     byte(qos),
	}
	s.sendMessage(msg)
}

func (s *ClientAndSession) Disconnect() error {
	if s.client.Properties.Clean {
		if s.isClose {
			return nil
		}
		s.Close()
		core.DelSession(s.info.deviceId)
	}
	return nil
}

func (s *ClientAndSession) Close() error {
	s.Lock()
	defer s.Unlock()

	if s.isClose {
		return nil
	}

	s.isClose = true
	close(s.done)
	core.DelSession(s.info.deviceId)
	return nil
}

func (s *ClientAndSession) SetDeviceId(deviceId string) {
	s.info.deviceId = deviceId
}

func (s *ClientAndSession) GetDeviceId() string {
	return s.info.deviceId
}

func (s *ClientAndSession) GetInfo() map[string]any {
	var protocolVersion = s.client.Properties.ProtocolVersion
	var protocolInfo = "MQTT 5.0"
	if protocolVersion < 5 {
		if protocolVersion == 4 {
			protocolInfo = "MQTT 3.1.1"
		} else {
			protocolInfo = "MQTT 3.1"
		}
	}
	return map[string]any{
		"clientId":     s.info.cid,
		"username":     s.info.username,
		"cleanStart":   s.client.State.Keepalive,
		"protocolInfo": protocolInfo,
		"deviceId":     s.info.deviceId,
		"topics":       s.info.Topics,
	}
}

func (s *ClientAndSession) sendMessage(msg *Message) {
	packet := packets.Packet{
		FixedHeader: packets.FixedHeader{
			Type: packets.Publish,
			Qos:  msg.QoS,
		},
		TopicName: msg.Topic,
		Payload:   msg.payload,
	}

	s.client.WritePacket(packet)
}
