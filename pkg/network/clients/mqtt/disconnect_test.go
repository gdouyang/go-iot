package mqttclient

import (
	"sync"
	"testing"

	MQTT "github.com/eclipse/paho.mqtt.golang"
	"github.com/stretchr/testify/require"
)

// noopPahoClient 仅用于测试 Disconnect 幂等性：除 Disconnect 外其它方法不会被触发。
type noopPahoClient struct{}

func (noopPahoClient) IsConnected() bool                       { return false }
func (noopPahoClient) IsConnectionOpen() bool                   { return false }
func (noopPahoClient) Connect() MQTT.Token                       { return nil }
func (noopPahoClient) Disconnect(quiesce uint)                   {}
func (noopPahoClient) Publish(topic string, qos byte, retained bool, payload interface{}) MQTT.Token {
	return nil
}
func (noopPahoClient) Subscribe(topic string, qos byte, callback MQTT.MessageHandler) MQTT.Token {
	return nil
}
func (noopPahoClient) SubscribeMultiple(filters map[string]byte, callback MQTT.MessageHandler) MQTT.Token {
	return nil
}
func (noopPahoClient) Unsubscribe(topics ...string) MQTT.Token { return nil }
func (noopPahoClient) AddRoute(topic string, callback MQTT.MessageHandler) {}
func (noopPahoClient) OptionsReader() MQTT.ClientOptionsReader  { return MQTT.ClientOptionsReader{} }

// 验证：MqttClientSession.Disconnect() 幂等。API Close、readLoop defer 与
// connectionLostHandler 可能并发触发；修复前 isClose 为普通 bool 无锁读写，并发
// 下可能多次 Disconnect 或并发写。
func TestMqttClientSession_DisconnectIdempotentAndConcurrent(t *testing.T) {
	s := &MqttClientSession{
		client:   noopPahoClient{},
		deviceId: "mci-0",
		choke:    make(chan MQTT.Message),
		done:     make(chan struct{}),
	}

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = s.Disconnect()
		}()
	}
	wg.Wait()

	require.True(t, s.isClose.Load(), "isClose should be set after Disconnect")
	require.NoError(t, s.Disconnect(), "Disconnect after close must be idempotent")
}
