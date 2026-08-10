package mqttserver

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"go-iot/pkg/core"
	"sync"

	logs "go-iot/pkg/logger"

	"github.com/eclipse/paho.mqtt.golang/packets"
)

type (
	// SessionInfo is info about session that will be put into etcd for persistency
	SessionInfo struct {
		// map subscribe topic to qos
		Username     string         `yaml:"name"` // 用户名
		Topics       map[string]int `yaml:"topics"`
		ClientID     string         `yaml:"clientID"`
		CleanFlag    bool           `yaml:"cleanFlag"`    // CleanSession
		ProtocolInfo string         `yaml:"protocolInfo"` //协议信息
		deviceId     string
	}

	// MqttSession includes the information about the connect between client and broker,
	// such as topic subscribe, not-send messages, etc.
	MqttSession struct {
		sync.Mutex
		broker       *Broker
		info         *SessionInfo
		done         chan struct{}
		pending      map[uint16]*Message
		pendingQueue []uint16
		nextID       uint16
		isClose      bool
		info1        map[string]any
	}

	// Message is the message send from broker to client
	Message struct {
		Topic      string `yaml:"topic"`
		B64Payload string `yaml:"b64Payload"`
		QoS        int    `yaml:"qos"`
	}
)

func newMsg(topic string, payload []byte, qos byte) *Message {
	m := &Message{
		Topic:      topic,
		B64Payload: base64.StdEncoding.EncodeToString(payload),
		QoS:        int(qos),
	}
	return m
}

func (s *MqttSession) init(b *Broker, connect *packets.ConnectPacket) error {
	s.broker = b
	s.done = make(chan struct{})
	s.pending = make(map[uint16]*Message)
	s.pendingQueue = []uint16{}

	s.info = &SessionInfo{}
	s.info.Username = connect.Username
	s.info.ClientID = connect.ClientIdentifier
	s.info.CleanFlag = true //connect.CleanSession not supported currently
	s.info.ProtocolInfo = fmt.Sprintf("%s %v", connect.ProtocolName, connect.ProtocolVersion)
	s.info.Topics = make(map[string]int)
	s.info1 = map[string]any{}

	go s.backgroundResendPending()

	return nil
}

func (s *MqttSession) subscribe(topics []string, qoss []byte) error {
	logs.Debugf("session %s sub %v", s.info.ClientID, topics)
	s.Lock()
	for i, t := range topics {
		s.info.Topics[t] = int(qoss[i])
	}
	s.Unlock()
	return nil
}

func (s *MqttSession) unsubscribe(topics []string) error {
	logs.Debugf("session %s unsub %v", s.info.ClientID, topics)
	s.Lock()
	for _, t := range topics {
		delete(s.info.Topics, t)
	}
	s.Unlock()
	return nil
}

func (s *MqttSession) getPacketFromMsg(topic string, payload []byte, qos byte) *packets.PublishPacket {
	p := packets.NewControlPacket(packets.Publish).(*packets.PublishPacket)
	p.Qos = qos
	p.TopicName = topic
	p.Payload = payload
	p.MessageID = s.nextID
	// the overflow is okay here
	// the session will give unique id from 0 to 65535 and do this again and again
	s.nextID++
	return p
}

// publishLocked 由调用方持 s.Lock 调用，内部不再取锁。选 qos 与写 pending 均在临界区内，
// 保证 topic-qos 读取与 nextID/pending 写入对 subscribe/unsubscribe 互斥；
// 否则 Publish 无锁读 s.info.Topics 与 subscribe 的 s.Lock 写促成 concurrent map read and map write。
func (s *MqttSession) publishLocked(topic string, payload []byte, qos byte) {
	client := s.broker.getClient(s.info.ClientID)
	if client == nil {
		logs.Errorf("client %s is offline in eg %v", s.info.ClientID, s.broker.productId)
		return
	}

	logs.Debugf("session %v publish %v", s.info.ClientID, topic)
	p := s.getPacketFromMsg(topic, payload, qos)
	if qos == QoS0 {
		select {
		case client.writeCh <- p:
		default:
		}
	} else if qos == QoS1 {
		msg := newMsg(topic, payload, qos)
		s.pending[p.MessageID] = msg
		s.pendingQueue = append(s.pendingQueue, p.MessageID)
		client.writePacket(p)
	} else {
		logs.Errorf("publish message with qos=2 is not supported currently")
	}
}

func (s *MqttSession) puback(p *packets.PubackPacket) {
	s.Lock()
	delete(s.pending, p.MessageID)
	s.Unlock()
}

// device session functions
func (s *MqttSession) Publish(topic string, payload string) {
	s.Lock()
	defer s.Unlock()
	qos, ok := s.info.Topics[topic]
	if !ok {
		qos = int(QoS0)
	}
	s.publishLocked(topic, []byte(payload), byte(qos))
}

func (s *MqttSession) PublishHex(topic string, payload string) {
	b, err := hex.DecodeString(payload)
	if err != nil {
		logs.Errorf("mqtt hex decode error: %v", err)
		return
	}
	s.Lock()
	defer s.Unlock()
	qos, ok := s.info.Topics[topic]
	if !ok {
		qos = int(QoS0)
	}
	s.publishLocked(topic, b, byte(qos))
}

func (s *MqttSession) Disconnect() error {
	if s.cleanSession() {
		if s.isClose {
			return nil
		}
		s.Close()
		core.DelSessionByUserDisconnect(s.info.deviceId)
	}
	return nil
}

func (s *MqttSession) Close() error {
	if s.cleanSession() {
		// 锁内检查+置位+close：readLoop 退出与重连踢旧会话可能并发调 Close，
		// 原实现无锁会 double close(s.done) panic
		s.Lock()
		if s.isClose {
			s.Unlock()
			return nil
		}
		s.isClose = true
		close(s.done)
		s.Unlock()
		logs.Debugf("session close %s", s.info.deviceId)
		client := s.broker.getClient(s.info.ClientID)
		if client != nil {
			client.close()
		}
	}
	return nil
}

func (s *MqttSession) SetDeviceId(deviceId string) {
	s.info.deviceId = deviceId
}
func (s *MqttSession) GetDeviceId() string {
	return s.info.deviceId
}
func (s *MqttSession) GetConInfo() map[string]any {
	// GetConInfo 每次 payload 写 s.info1：与其它并发 GetConInfo/API 查询不持锁会
	// fatal: concurrent map read and map write 清写共享 map
	s.Lock()
	defer s.Unlock()
	s.info1["username"] = s.info.Username
	s.info1["clientID"] = s.info.ClientID
	s.info1["cleanSession"] = s.info.CleanFlag
	s.info1["protocol"] = s.info.ProtocolInfo
	return s.info1
}
