package server

import (
	"container/list"
	"go-iot/provider/util"
	"time"

	"github.com/beego/beego/v2/core/logs"
	"github.com/gorilla/websocket"
)

type EventType int

const (
	EVENT_JOIN = iota
	EVENT_LEAVE
	EVENT_MESSAGE

	ECHO  = "echo"  // 浏览器使用
	NORTH = "north" // 北向接口使用
)

type Event struct {
	Type      EventType // JOIN, LEAVE, MESSAGE
	Addr      string
	Timestamp int // Unix timestamp (secs)
	Content   string
}

func PushNorth(msg string) {
	publish <- newEvent(EVENT_MESSAGE, msg)
}

// 创建事件
func newEvent(ep EventType, msg string) Event {
	return Event{ep, "", int(time.Now().Unix()), msg}
}

// 加入
func Join(type_ string, evt string, ws *websocket.Conn) string {
	if len(type_) == 0 {
		logs.Error("type can not be null echo or north")
	}
	addr := ws.RemoteAddr().String()
	subscribe <- Subscriber{Type: type_, Evt: evt, Addr: addr, Conn: ws}

	return addr
}

// 离开
func Leave(id string) {
	unsubscribe <- id
}

// 订阅者
type Subscriber struct {
	Type string // north\echo
	Evt  string // 客户端订阅的事件
	Addr string
	Conn *websocket.Conn // Only for WebSocket users; otherwise nil.
}

var (
	// Channel for new join users.
	subscribe = make(chan Subscriber, 10)
	// Channel for exit users.
	unsubscribe = make(chan string, 10)
	// Send events here to publish them.
	publish         = make(chan Event, 10)
	subscribers     = list.New()
	echoSubscribers = list.New()
)

// This function handles all incoming chan messages.
func chatroom() {
	for {
		select {
		case sub := <-subscribe:
			switch sub.Type {
			case ECHO:
				echoSubscribers.PushBack(sub) // Add user to the end of list.
				logs.Info("New echo user:", sub.Addr, ";WebSocket:", sub.Conn != nil)
			case NORTH:
				subscribers.PushBack(sub) // Add user to the end of list.
				logs.Info("New north user:", sub.Addr, ";WebSocket:", sub.Conn != nil)
			default:
				Leave(sub.Addr)
				logs.Error("Type not persent(echo or north)")
			}
		case event := <-publish:
			broadcastWebSocket(event)

			if event.Type == EVENT_MESSAGE {
				logs.Info("Message from", event.Addr, ";Content:", event.Content)
			}
		case unsub := <-unsubscribe:
			for sub := subscribers.Front(); sub != nil; sub = sub.Next() {
				if sub.Value.(Subscriber).Addr == unsub {
					subscribers.Remove(sub)
					// Clone connection.
					ws := sub.Value.(Subscriber).Conn
					if ws != nil {
						ws.Close()
						logs.Error("NorthWebSocket closed:", unsub)
					}
					// publish <- newEvent(models.EVENT_LEAVE, unsub, "", ) // Publish a LEAVE event.
					break
				}
			}
			for sub := echoSubscribers.Front(); sub != nil; sub = sub.Next() {
				if sub.Value.(Subscriber).Addr == unsub {
					echoSubscribers.Remove(sub)
					// Clone connection.
					ws := sub.Value.(Subscriber).Conn
					if ws != nil {
						ws.Close()
						logs.Error("EchoWebSocket closed:", unsub)
					}
					break
				}
			}
		}
	}
}

// 广播发送给WebSocket用户
func broadcastWebSocket(event Event) {
	data, err := util.JsonEncoderHTML(event)
	if err != nil {
		logs.Error("Fail to marshal event:", err)
		return
	}

	if subscribers.Len() < 1 {
		EchoToBrower(EchoMsg{Msg: "无NorthWebSocket订阅:" + event.Content, Type: "northws"})
		return
	}

	for sub := subscribers.Front(); sub != nil; sub = sub.Next() {
		// Immediately send event to WebSocket users.
		suber := sub.Value.(Subscriber)
		ws := suber.Conn
		if ws != nil {
			if ws.WriteMessage(websocket.TextMessage, data) == nil {
				content := "向[" + suber.Addr + "]推送:" + event.Content
				EchoToBrower(EchoMsg{Msg: content, Type: "northws"})
			} else {
				// User disconnected.
				unsubscribe <- sub.Value.(Subscriber).Addr
			}
		}
	}
}

type EchoMsg struct {
	Msg  string `json:"msg"`
	Type string `json:"type"`
}

// 北向接口消息输出到浏览器
func EchoToBrower(msg EchoMsg) {
	bytedata, _ := util.JsonEncoderHTML(msg)
	for sub := echoSubscribers.Front(); sub != nil; sub = sub.Next() {
		// Immediately send event to WebSocket users.
		ws := sub.Value.(Subscriber).Conn
		if ws != nil {
			if ws.WriteMessage(websocket.TextMessage, bytedata) != nil {
				// User disconnected.
				unsubscribe <- sub.Value.(Subscriber).Addr
			}
		}
	}
}

func init() {
	go chatroom()
}
