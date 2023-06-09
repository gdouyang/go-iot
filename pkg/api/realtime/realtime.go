package realtime

import (
	"container/list"
	"encoding/json"
	"go-iot/pkg/boot"
	"go-iot/pkg/cluster"
	"go-iot/pkg/eventbus"
	"go-iot/pkg/redis"
	"sync"

	"github.com/gorilla/websocket"
)

func init() {
	boot.AddStartLinstener(func() {
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
func ListenEventBus(topic string) {
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
	subscribers: sync.Map{},
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

type realtime struct {
	subscribe   chan Subscriber
	unsubscribe chan Subscriber
	publish     chan eventbus.Message
	subscribers sync.Map //map[string]*list.List
}

func (e *realtime) getSubscriber(deviceId string) (*list.List, bool) {
	val, ok := e.subscribers.Load(deviceId)
	if ok {
		if val != nil {
			return val.(*list.List), ok
		}
		return nil, ok
	}
	return nil, false
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
			val, ok := e.getSubscriber(sub.DeviceId)
			if !ok {
				val = list.New()
				e.subscribers.Store(sub.DeviceId, val)
			}
			val.PushBack(&sub)
		case event := <-e.publish:
			subs, _ := e.getSubscriber(event.GetDeviceId())
			if subs != nil {
				for sub := subs.Front(); sub != nil; sub = sub.Next() {
					suber := sub.Value.(*Subscriber)
					ws := suber.Conn
					if ws != nil {
						d, _ := json.Marshal(event)
						ws.WriteMessage(websocket.TextMessage, d)
						if event.Type() != clusterMessageType && cluster.Enabled() {
							clusterMsg := clusterMessage{ClusterId: cluster.GetClusterId(), DeviceId: event.GetDeviceId(), Origin: string(d)}
							clusterMsgStr, _ := json.Marshal(clusterMsg)
							redis.Pub(_CLUST_EVENT_KEY, clusterMsgStr)
						}
					}
				}
			}
		case unsub := <-e.unsubscribe:
			subs, _ := e.getSubscriber(unsub.DeviceId)
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
