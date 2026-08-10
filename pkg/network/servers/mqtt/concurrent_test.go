package mqttserver

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/eclipse/paho.mqtt.golang/packets"
)

// 验证：MqttSession.GetConInfo() 并发调用（旧实现每次写共享 s.info1 map，在 API
// 多请求/上下线并发查询连接信息时会触发 fatal: concurrent map read and map write）。
// 修复：GetConInfo 持 s.Lock 写 s.info1。
func TestMqttSession_GetConInfoConcurrentSafe(t *testing.T) {
	s := &MqttSession{
		info:  &SessionInfo{ClientID: "c1", Username: "u1", CleanFlag: true, ProtocolInfo: "MQTT 3.1.1"},
		info1: map[string]any{},
	}

	var stop atomic.Bool
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for !stop.Load() {
				_ = s.GetConInfo()
			}
		}()
	}
	time.Sleep(1 * time.Second)
	stop.Store(true)
	wg.Wait()
}

// 验证：MqttSession.Publish/PublishHex 并发与 subscribe/unsubscribe 竞争读写
// s.info.Topics（subscribe 在 s.Lock 下写，Publish 原先无锁读 s.info.Topics）
// 会触发 fatal: concurrent map read and map write。
// 修复：Publish/PublishHex 持 s.Lock，publish 改为 publishLocked（不再取锁）。
func TestMqttSession_PublishConcurrentWithSubscribeSafe(t *testing.T) {
	b := &Broker{productId: "p1", clients: map[string]*Client{}}
	c := &Client{
		info:    ClientInfo{cid: "c1"},
		writeCh: make(chan packets.ControlPacket, 50),
		done:    make(chan struct{}),
	}
	b.clients["c1"] = c

	s := &MqttSession{
		broker: b,
		info: &SessionInfo{
			ClientID: "c1",
			Topics:   map[string]int{"t/**": 0},
		},
		info1:        map[string]any{},
		pending:      make(map[uint16]*Message),
		pendingQueue: []uint16{},
		done:         make(chan struct{}),
	}

	var stop atomic.Bool
	var wg sync.WaitGroup

	// 并发 PubSub：subscribe/unsubscribe 写 s.info.Topics，Publish/PublishHex 读 s.info.Topics
	wg.Add(1)
	go func() {
		defer wg.Done()
		on := true
		for !stop.Load() {
			if on {
				s.subscribe([]string{"a/b", "c/d"}, []byte{1, 0})
			} else {
				s.unsubscribe([]string{"a/b", "c/d"})
			}
			on = !on
		}
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		for !stop.Load() {
			s.Publish("t/**", "hello")
		}
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		for !stop.Load() {
			s.PublishHex("t/**", "68656c6c6f")
		}
	}()

	time.Sleep(1 * time.Second)
	stop.Store(true)
	wg.Wait()
}
