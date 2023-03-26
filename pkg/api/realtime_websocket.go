package api

import (
	"container/list"
	"encoding/json"
	"errors"
	"fmt"
	"go-iot/pkg/core/eventbus"
	device "go-iot/pkg/models/device"
	"net/http"
	"sync"

	"github.com/beego/beego/v2/core/logs"
	"github.com/beego/beego/v2/server/web"
	"github.com/gorilla/websocket"
)

// websocket实时信息，把监听的数据通过websocket返回
func init() {
	web.Router("/api/realtime/:deviceId/:type", &RealtimeWebSocketController{}, "get:Join")

	go realtimeInstance.writeLoop()
}

type RealtimeWebSocketController struct {
	AuthController
}

// 加入方法
func (ctl *RealtimeWebSocketController) Join() {
	deviceId := ctl.Param(":deviceId")
	if len(deviceId) == 0 {
		ctl.RespError(errors.New("deviceId must be present"))
		return
	}
	typ := ctl.Param(":type")
	if len(typ) == 0 {
		ctl.RespError(errors.New("type must be present"))
		return
	}
	dev, err := device.GetDeviceMust(deviceId)
	if err != nil {
		ctl.RespError(err)
		return
	}
	// Upgrade from http request to WebSocket.
	var upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	} // use default options
	ws, err := upgrader.Upgrade(ctl.Ctx.ResponseWriter, ctl.Ctx.Request, nil)
	if _, ok := err.(websocket.HandshakeError); ok {
		http.Error(ctl.Ctx.ResponseWriter, "Not a websocket handshake", 400)
		return
	} else if err != nil {
		logs.Error("Cannot setup WebSocket connection:", err)
		ctl.RespError(fmt.Errorf("cannot setup WebSocket connection:%v", err))
		return
	}

	// Join.
	addr := ws.RemoteAddr().String()
	topic := fmt.Sprintf("/device/%s/%s/%s", dev.ProductId, dev.Id, typ)
	realtimeInstance.subscribe <- subscriber{ProductId: dev.ProductId, DeviceId: deviceId, topic: topic, Addr: addr, Conn: ws}
	defer func() {
		realtimeInstance.unsubscribe <- subscriber{ProductId: dev.ProductId, DeviceId: deviceId, topic: topic, Addr: addr}
	}()

	eventbus.Subscribe(topic, realtimeInstance.send)

	// Message receive loop.
	for {
		_, _, err := ws.ReadMessage()
		if err != nil {
			if web.BConfig.WebConfig.AutoRender {
				ctl.RespOk()
			}
			return
		}
	}
}

// 实例
var realtimeInstance *realtime = &realtime{
	subscribe: make(chan subscriber, 10),
	// Channel for exit users.
	unsubscribe: make(chan subscriber, 10),
	// Send events here to publish them.
	publish:     make(chan eventbus.Message, 10),
	subscribers: sync.Map{},
}

type realtime struct {
	subscribe   chan subscriber
	unsubscribe chan subscriber
	publish     chan eventbus.Message
	subscribers sync.Map //map[string]*list.List
}

// 订阅者
type subscriber struct {
	ProductId string
	DeviceId  string
	topic     string
	Addr      string
	Conn      *websocket.Conn // Only for WebSocket users; otherwise nil.
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
func (e *realtime) send(msg eventbus.Message) {
	e.publish <- msg
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
					suber := sub.Value.(*subscriber)
					ws := suber.Conn
					if ws != nil {
						d, _ := json.Marshal(event)
						ws.WriteMessage(websocket.TextMessage, d)
					}
				}
			}
		case unsub := <-e.unsubscribe:
			subs, _ := e.getSubscriber(unsub.DeviceId)
			if subs != nil {
				for sub := subs.Front(); sub != nil; sub = sub.Next() {
					suber := sub.Value.(*subscriber)
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
					eventbus.UnSubscribe(unsub.topic, e.send)
				}
			}
		}
	}
}
