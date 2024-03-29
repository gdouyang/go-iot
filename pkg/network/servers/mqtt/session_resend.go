package mqttserver

import (
	"encoding/base64"
	"time"

	logs "go-iot/pkg/logger"

	"github.com/eclipse/paho.mqtt.golang/packets"
)

var suppertMqttQOS2 bool = false

func (s *MqttSession) cleanSession() bool {
	if !suppertMqttQOS2 {
		return true
	}
	return s.info.CleanFlag
}

func (s *MqttSession) backgroundResendPending() {
	if !suppertMqttQOS2 {
		return
	}
	debugLogTime := time.Now().Add(time.Minute)
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-s.done:
			return
		case <-ticker.C:
			s.doResend()
		}
		if time.Now().After(debugLogTime) {
			logs.Debugf("session %v resend", s.info.ClientID)
			debugLogTime = time.Now().Add(time.Minute)
		}
	}
}

func (s *MqttSession) doResend() {
	client := s.broker.getClient(s.info.ClientID)
	s.Lock()
	defer s.Unlock()

	if len(s.pending) == 0 {
		s.pendingQueue = []uint16{}
		return
	}
	for i, idx := range s.pendingQueue {
		if val, ok := s.pending[idx]; ok {
			// find first msg need to resend
			s.pendingQueue = s.pendingQueue[i:]
			p := packets.NewControlPacket(packets.Publish).(*packets.PublishPacket)
			p.Qos = byte(val.QoS)
			p.TopicName = val.Topic
			payload, err := base64.StdEncoding.DecodeString(val.B64Payload)
			if err != nil {
				logs.Errorf("base64 decode error for Message B64Payload %v", err)
				return
			}
			p.Payload = payload
			p.MessageID = idx
			if client != nil {
				client.writePacket(p)
			} else {
				logs.Debugf("session %v do resend but client is nil", s.info.ClientID)
			}
			return
		}
	}
}
