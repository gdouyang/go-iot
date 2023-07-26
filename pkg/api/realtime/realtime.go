package realtime

import (
	"container/list"
	"encoding/json"
	"fmt"
	"go-iot/pkg/boot"
	"go-iot/pkg/cluster"
	"go-iot/pkg/eventbus"
	"go-iot/pkg/redis"

	"github.com/gorilla/websocket"
)

func init() {
	boot.AddStartLinstener(func() {
		go listenEventBus("/device/*/*/*")
		go realtimeInstance.writeLoop()
		go realtimeInstance.listenerCluster()
	})
}

// 订阅消息
func Subscribe(sub Subscriber) {
	realtimeInstance.subscribe <- sub
}

// 取消订阅
func Unsubscribe(sub Subscriber) {
	realtimeInstance.unsubscribe <- sub
}

// 监听事件总线
func listenEventBus(topic string) {
	eventbus.Subscribe(topic, send)
}

// 发送消息
func send(msg eventbus.Message) {
	realtimeInstance.publish <- msg
}

// 实例
var realtimeInstance *realtime = &realtime{
	subscribe: make(chan Subscriber, 10),
	// Channel for exit users.
	unsubscribe: make(chan Subscriber, 10),
	// Send events here to publish them.
	publish:     make(chan eventbus.Message, 10),
	subscribers: list.New(),
	matcher:     eventbus.NewAntPathMatcher(),
}

// 订阅者
type Subscriber struct {
	ProductId string
	DeviceId  string
	Topic     string
	Addr      string
	Conn      *websocket.Conn // Only for WebSocket users; otherwise nil.
}

// 集群realtime消息
type clusterMessage struct {
	ClusterId string `json:"clusterId"`
	DeviceId  string `json:"deviceId"`
	ProductId string `json:"productId"`
	Origin    string `json:"origin"`
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

type realtime struct {
	subscribe   chan Subscriber
	unsubscribe chan Subscriber
	publish     chan eventbus.Message
	subscribers *list.List
	matcher     *eventbus.AntPathMatcher
}

const _CLUST_EVENT_KEY = "go:cluster:realtime"

func (e *realtime) listenerCluster() {
	if cluster.Enabled() {
		for msg := range redis.Sub(_CLUST_EVENT_KEY) {
			payload := msg.Payload
			var clusterMsg clusterMessage
			json.Unmarshal([]byte(payload), &clusterMsg)
			// 不是同一个节点的数据才接收
			if len(clusterMsg.ClusterId) > 0 && clusterMsg.ClusterId != cluster.GetClusterId() {
				send(&clusterMsg)
			}
		}
	}
}

func (e *realtime) writeLoop() {
	for {
		select {
		case sub := <-e.subscribe:
			e.subscribers.PushBack(&sub)
		case event := <-e.publish:
			subs := e.subscribers
			if subs != nil {
				for sub := subs.Front(); sub != nil; sub = sub.Next() {
					suber := sub.Value.(*Subscriber)
					// 这里只处理满足条件的
					path := fmt.Sprintf("/device/%s/%s/%s", event.GetProductId(), event.GetDeviceId(), event.Type())
					if !e.matcher.Match(suber.Topic, path) {
						continue
					}
					ws := suber.Conn
					if ws != nil {
						d, _ := json.Marshal(event)
						ws.WriteMessage(websocket.TextMessage, d)
						if event.Type() != clusterMessageType && cluster.Enabled() {
							clusterMsg := clusterMessage{ClusterId: cluster.GetClusterId(), DeviceId: event.GetDeviceId(), ProductId: event.GetProductId(), Origin: string(d)}
							clusterMsgStr, _ := json.Marshal(clusterMsg)
							redis.Pub(_CLUST_EVENT_KEY, clusterMsgStr)
						}
					}
				}
			}
		case unsub := <-e.unsubscribe:
			subs := e.subscribers
			if subs != nil {
				for sub := subs.Front(); sub != nil; sub = sub.Next() {
					suber := sub.Value.(*Subscriber)
					if suber.Addr == unsub.Addr {
						subs.Remove(sub)
						ws := suber.Conn
						if ws != nil {
							ws.Close()
						}
						break
					}
				}
				if subs.Len() == 0 {
					eventbus.UnSubscribe(unsub.Topic, send)
				}
			}
		}
	}
}
