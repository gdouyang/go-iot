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

		broker      *Broker
		client      *mqtt.Client
		info        ClientInfo
		isClose     bool
		done        chan struct{}
		connectInfo map[string]any
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

	var newSession *ClientAndSession
	oldSession := core.GetSession(cl.ID)
	if oldSession != nil {
		newSession = oldSession.(*ClientAndSession)
		newSession.client = cl
		newSession.info = info
		newSession.done = make(chan struct{})
		newSession.isClose = false
	} else {
		newSession = &ClientAndSession{
			broker:      broker,
			client:      cl,
			info:        info,
			done:        make(chan struct{}),
			connectInfo: map[string]any{},
		}
	}

	return newSession
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
	core.DelSessionByUserDisconnect(s.info.deviceId)
	s.Close()
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
	if !s.client.Closed() {
		s.broker.server.DisconnectClient(s.client, packets.CodeDisconnect)
	}
	return nil
}

// closeByBroker 由 broker 的 OnDisconnect 回调调用（broker 锁已持有）：
// 只关闭 session 本地资源，不调用 DisconnectClient——客户端已处于断开流程，
// DisconnectClient 会同步回调 OnDisconnect 造成 broker 锁重入死锁。
// 用 TryLock：OnDisconnect 可能被 DisconnectClient 同步触发（调用者已持有本锁），
// 拿不到锁说明关闭流程由锁持有方（Close/Disconnect）负责，跳过即可。
func (s *ClientAndSession) closeByBroker() {
	if !s.TryLock() {
		return
	}
	defer s.Unlock()
	if s.isClose {
		return
	}
	s.isClose = true
	close(s.done)
}

func (s *ClientAndSession) SetDeviceId(deviceId string) {
	s.info.deviceId = deviceId
}

func (s *ClientAndSession) GetDeviceId() string {
	return s.info.deviceId
}

func (s *ClientAndSession) GetConInfo() map[string]any {
	var protocolVersion = s.client.Properties.ProtocolVersion
	var protocolInfo = "MQTT 5.0"
	if protocolVersion < 5 {
		if protocolVersion == 4 {
			protocolInfo = "MQTT 3.1.1"
		} else {
			protocolInfo = "MQTT 3.1"
		}
	}
	s.connectInfo["clientId"] = s.info.cid
	s.connectInfo["username"] = s.info.username
	s.connectInfo["cleanStart"] = s.client.State.Keepalive
	s.connectInfo["protocolInfo"] = protocolInfo
	s.connectInfo["deviceId"] = s.info.deviceId
	s.connectInfo["topics"] = s.info.Topics

	return s.connectInfo
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

	err := s.client.WritePacket(packet)
	if err != nil {
		logs.Warnf("sendMessage error: %s", err.Error())
	}
}
