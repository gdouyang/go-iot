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
		cid       string
		username  string
		password  string
		deviceId  string
		productId string
		Topics    map[string]byte
	}

	// ClientAndSession represents a MQTT5 client connection
	ClientAndSession struct {
		sync.Mutex

		broker      *Broker
		client      *mqtt.Client
		info        ClientInfo
		infoMu      sync.Mutex // 保护 info（特别是 deviceId）读写：NewClient 接管替换整块 info 会
		           // 与并发 OnPublished→GetDeviceId race；与 broker.Lock 锁序独立
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
		// infoMu 内：接管替换整块 info 与并发 GetDeviceId/GetConInfo race；保留旧
		// deviceId（原实现 newSession.info = info 会覆盖 SetDeviceId 设进的 deviceId）。
		newSession.infoMu.Lock()
		if len(newSession.info.deviceId) > 0 {
			info.deviceId = newSession.info.deviceId
		}
		newSession.client = cl
		newSession.info = info
		newSession.done = make(chan struct{})
		newSession.isClose = false
		newSession.infoMu.Unlock()
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
	c.infoMu.Lock()
	defer c.infoMu.Unlock()
	return c.info.cid
}

func (c *ClientAndSession) UserName() string {
	c.infoMu.Lock()
	defer c.infoMu.Unlock()
	return c.info.username
}

func (c *ClientAndSession) Done() <-chan struct{} {
	c.infoMu.Lock()
	defer c.infoMu.Unlock()
	return c.done
}

// device session functions
func (s *ClientAndSession) Publish(topic string, payload string) {
	s.infoMu.Lock()
	qos, ok := s.info.Topics[topic]
	s.infoMu.Unlock()
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
	s.infoMu.Lock()
	qos, ok := s.info.Topics[topic]
	s.infoMu.Unlock()
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
	s.infoMu.Lock()
	did := s.info.deviceId
	s.infoMu.Unlock()
	core.DelSessionByUserDisconnect(did)
	s.Close()
	return nil
}

func (s *ClientAndSession) Close() error {
	s.Lock()
	defer s.Unlock()

	// isClose/done/client 与 info 字段统一由 infoMu 保护，与 NewClient 接管路径保持同一把锁；
	// s.Lock 仅作为 Close 与 closeByBroker 之间的重入护栏。
	// DisconnectClient 必须在释放 infoMu 之后调用：其同步触发的 OnDisconnect 会获取 broker.Lock，
	// 而 OnConnectAuthenticate 持有 broker.Lock 等待 infoMu，持锁调用将形成锁序反转。
	s.infoMu.Lock()
	if s.isClose {
		s.infoMu.Unlock()
		return nil
	}
	s.isClose = true
	close(s.done)
	client := s.client
	s.infoMu.Unlock()

	if !client.Closed() {
		s.broker.server.DisconnectClient(client, packets.CodeDisconnect)
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
	s.infoMu.Lock()
	defer s.infoMu.Unlock()
	if s.isClose {
		return
	}
	s.isClose = true
	close(s.done)
}

func (s *ClientAndSession) SetDeviceId(deviceId string) {
	s.infoMu.Lock()
	defer s.infoMu.Unlock()
	s.info.deviceId = deviceId
}

func (s *ClientAndSession) GetDeviceId() string {
	s.infoMu.Lock()
	defer s.infoMu.Unlock()
	return s.info.deviceId
}

func (s *ClientAndSession) GetConInfo() map[string]any {
	// GetConInfo 写共享 s.connectInfo：与并发 API 查询/其它 GetConInfo 不持锁会
	// fatal: concurrent map read and map write
	s.Lock()
	defer s.Unlock()
	s.infoMu.Lock()
	defer s.infoMu.Unlock()
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
	s.connectInfo["productId"] = s.info.productId
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
