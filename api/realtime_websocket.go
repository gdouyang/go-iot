package api

import (
	"container/list"
	"encoding/json"
	"errors"
	"fmt"
	"go-iot/codec/eventbus"
	device "go-iot/models/device"
	"net/http"
	"sync"

	"github.com/beego/beego/v2/core/logs"
	"github.com/beego/beego/v2/server/web"
	"github.com/gorilla/websocket"
)

func init() {
	web.Router("/api/realtime/:deviceId/:type", &RealtimeWebSocketController{}, "get:Join")

	go writeLoop()
}

// 订阅者
type subscriber struct {
	ProductId string
	DeviceId  string
	topic     string
	Addr      string
	Conn      *websocket.Conn // Only for WebSocket users; otherwise nil.
}

var (
	// Channel for new join users.
	subscribe = make(chan subscriber, 10)
	// Channel for exit users.
	unsubscribe = make(chan subscriber, 10)
	// Send events here to publish them.
	publish     = make(chan eventbus.Message, 10)
	subscribers = map[string]*list.List{}
)

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
	ws, err := websocket.Upgrade(ctl.Ctx.ResponseWriter, ctl.Ctx.Request, nil, 1024, 1024)
	if _, ok := err.(websocket.HandshakeError); ok {
		http.Error(ctl.Ctx.ResponseWriter, "Not a websocket handshake", 400)
		return
	} else if err != nil {
		logs.Error("Cannot setup WebSocket connection:", err)
		return
	}

	// Join.
	addr := ws.RemoteAddr().String()
	topic := fmt.Sprintf("/device/%s/%s/%s", dev.ProductId, dev.Id, typ)
	subscribe <- subscriber{ProductId: dev.ProductId, DeviceId: deviceId, topic: topic, Addr: addr, Conn: ws}
	defer func() {
		unsubscribe <- subscriber{ProductId: dev.ProductId, DeviceId: deviceId, topic: topic, Addr: addr}
	}()

	eventbus.Subscribe(topic, send)

	// Message receive loop.
	for {
		_, _, err := ws.ReadMessage()
		if err != nil {
			return
		}
	}
}

func send(msg eventbus.Message) {
	publish <- msg
}

func writeLoop() {
	for {
		select {
		case sub := <-subscribe:
			l := sync.Mutex{}
			l.Lock()
			defer l.Unlock()
			if _, ok := subscribers[sub.DeviceId]; !ok {
				subscribers[sub.DeviceId] = list.New()
			}
			subscribers[sub.DeviceId].PushBack(&sub)
		case event := <-publish:
			subs := subscribers[event.GetDeviceId()]
			for sub := subs.Front(); sub != nil; sub = sub.Next() {
				suber := sub.Value.(*subscriber)
				ws := suber.Conn
				if ws != nil {
					d, _ := json.Marshal(event)
					ws.WriteMessage(websocket.TextMessage, d)
				}
			}
		case unsub := <-unsubscribe:
			subs := subscribers[unsub.DeviceId]
			for sub := subs.Front(); sub != nil; sub = sub.Next() {
				suber := sub.Value.(*subscriber)
				if suber.Addr == unsub.Addr {
					subs.Remove(sub)
					// Clone connection.
					ws := suber.Conn
					if ws != nil {
						ws.Close()
						logs.Error("NorthWebSocket closed:", unsub)
					}
					break
				}
			}
			if subs.Len() == 0 {
				eventbus.UnSubscribe(unsub.topic, send)
			}
		}
	}
}
