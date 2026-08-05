// Package eventpush 将设备相关 eventbus 消息扇出到管理端 WebSocket 订阅者；
// 集群模式下经 Redis 频道 go:cluster:eventpush 同步到其它节点。
package eventpush

import (
	"container/list"
	"encoding/json"
	"fmt"
	"go-iot/pkg/cluster"
	"go-iot/pkg/eventbus"
	"go-iot/pkg/redis"
	"sync"

	"github.com/gorilla/websocket"

	logs "go-iot/pkg/logger"
)

// ClusterChannel 集群事件推送 Redis Pub/Sub 频道。
const ClusterChannel = "go:cluster:eventpush"

var startOnce sync.Once

// Start 启动推送循环（eventbus 订阅、写循环、集群 Redis 订阅）。
// 由 app.Start 显式调用；可重复调用，仅生效一次。
func Start() {
	startOnce.Do(func() {
		go listenEventBus("/device/*/*/*")
		go hub.writeLoop()
		go hub.listenerCluster()
		logs.Infof("eventpush started")
	})
}

// Subscribe 注册 WebSocket 订阅者。
func Subscribe(sub Subscriber) {
	hub.subscribe <- sub
}

// Unsubscribe 取消订阅。
func Unsubscribe(sub Subscriber) {
	hub.unsubscribe <- sub
}

func listenEventBus(topic string) {
	eventbus.Subscribe(topic, send)
}

func send(msg eventbus.Message) {
	hub.publish <- msg
}

var hub = &eventHub{
	subscribe:   make(chan Subscriber, 10),
	unsubscribe: make(chan Subscriber, 10),
	publish:     make(chan eventbus.Message, 10),
	subscribers: list.New(),
	matcher:     eventbus.NewAntPathMatcher(),
}

// Subscriber WebSocket 订阅者。
type Subscriber struct {
	ProductId string
	DeviceId  string
	Topic     string
	Addr      string
	Conn      *websocket.Conn
}

// 集群间转发的事件包装
type clusterMessage struct {
	ClusterId  string               `json:"clusterId"`
	DeviceId   string               `json:"deviceId"`
	ProductId  string               `json:"productId"`
	OriginType eventbus.MessageType `json:"originType"`
	Origin     string               `json:"origin"`
}

func (m *clusterMessage) MarshalJSON() ([]byte, error) {
	return []byte(m.Origin), nil
}

const clusterMessageType eventbus.MessageType = eventbus.MessageType("cluster")

func (m *clusterMessage) Type() eventbus.MessageType {
	return clusterMessageType
}

func (m *clusterMessage) GetDeviceId() string {
	return m.DeviceId
}
func (m *clusterMessage) GetProductId() string {
	return m.ProductId
}

type eventHub struct {
	subscribe   chan Subscriber
	unsubscribe chan Subscriber
	publish     chan eventbus.Message
	subscribers *list.List
	matcher     *eventbus.AntPathMatcher
}

func (e *eventHub) listenerCluster() {
	if cluster.Enabled() {
		for msg := range redis.Sub(ClusterChannel) {
			payload := msg.Payload
			var clusterMsg clusterMessage
			json.Unmarshal([]byte(payload), &clusterMsg)
			if len(clusterMsg.ClusterId) > 0 && clusterMsg.ClusterId != cluster.GetClusterId() {
				send(&clusterMsg)
			}
		}
	}
}

func (e *eventHub) writeLoop() {
	for {
		select {
		case sub := <-e.subscribe:
			e.subscribers.PushBack(&sub)
		case event := <-e.publish:
			e.dispatch(event)
		case unsub := <-e.unsubscribe:
			e.removeByAddr(unsub.Addr)
		}
	}
}

func (e *eventHub) dispatch(event eventbus.Message) {
	if e.subscribers == nil {
		return
	}
	path := eventPath(event)
	payload, _ := json.Marshal(event)
	clusterFanout := event.Type() != clusterMessageType && cluster.Enabled()

	for el := e.subscribers.Front(); el != nil; el = el.Next() {
		sub := el.Value.(*Subscriber)
		if sub.Conn == nil || !e.matcher.Match(sub.Topic, path) {
			continue
		}
		_ = sub.Conn.WriteMessage(websocket.TextMessage, payload)
		if clusterFanout {
			publishClusterEvent(event, payload)
			// 每个本机事件只需 Pub 一次，避免按订阅者重复广播
			clusterFanout = false
		}
	}
}

func eventPath(event eventbus.Message) string {
	eventType := event.Type()
	if t, ok := event.(*clusterMessage); ok {
		eventType = t.OriginType
	}
	return fmt.Sprintf("/device/%s/%s/%s", event.GetProductId(), event.GetDeviceId(), eventType)
}

func publishClusterEvent(event eventbus.Message, origin []byte) {
	msg := clusterMessage{
		ClusterId:  cluster.GetClusterId(),
		DeviceId:   event.GetDeviceId(),
		ProductId:  event.GetProductId(),
		OriginType: event.Type(),
		Origin:     string(origin),
	}
	b, _ := json.Marshal(msg)
	redis.Pub(ClusterChannel, b)
}

func (e *eventHub) removeByAddr(addr string) {
	if e.subscribers == nil {
		return
	}
	for el := e.subscribers.Front(); el != nil; el = el.Next() {
		sub := el.Value.(*Subscriber)
		if sub.Addr != addr {
			continue
		}
		e.subscribers.Remove(el)
		if sub.Conn != nil {
			_ = sub.Conn.Close()
		}
		return
	}
}
