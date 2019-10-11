package xixun

import (
	"encoding/json"
	"fmt"
	"go-iot/provider/utils"
	"log"
	"net/http"
	"sync"

	"github.com/astaxie/beego"
	"github.com/gorilla/websocket"
)

type XixunLED struct {
	uid  int             //WebSocket连接唯一标识
	SN   string          //设备SN
	Conn *websocket.Conn // websocket连接
	Cond *sync.Cond      // 同步调用的condition
	Resp string          // 命令返回
}

// 心跳
type breath struct {
	Sn string `json:"cardId"`
}

var (
	subscribers = map[string]XixunLED{}
)

// 启动WebSocket
func startWebSocket() {
	port := beego.AppConfig.DefaultInt("xixunport", 7078)
	go (func() {
		beego.Info(fmt.Sprintf("xixun WebSocket server Running on :%d", port))
		http.HandleFunc("/", upgradeWs)
		err := http.ListenAndServe(fmt.Sprintf("%s:%d", "0.0.0.0", port), nil)
		if err != nil {
			beego.Error("xixun WebSocket server Start error:", err)
		}
	})()
}

// 把http升级为websocket
func upgradeWs(w http.ResponseWriter, r *http.Request) {
	c, err := websocket.Upgrade(w, r, nil, 1024, 1024)
	if err != nil {
		beego.Error("upgrade fail:", err)
		return
	}

	var sn string

	var l sync.Mutex
	led := XixunLED{uid: utils.Uuid(), SN: sn, Conn: c, Cond: sync.NewCond(&l)}
	for {
		mt, message, err := c.ReadMessage()
		if err != nil {
			beego.Error("ws read err:", err)
			break
		}
		resp := string(message)
		beego.Info("mssageType:", mt, "message :", resp)
		abc := breath{}
		json.Unmarshal(message, &abc)
		sn = abc.Sn
		if len(sn) > 0 {
			beego.Info("breath ----->", sn)
			_, ok := subscribers[sn]
			if !ok {
				led.SN = sn
				subscribers[sn] = led
				beego.Info("led connected, connection len:", len(subscribers))
			}
		} else {
			beego.Info("response ----->", led.SN)
			led, ok := subscribers[led.SN]
			if ok {

				led.Cond.L.Lock()
				rs := &led.Resp
				*rs = string(message)
				beego.Info("do response", &led.Resp, led.Resp)
				led.Cond.Signal()
				led.Cond.L.Unlock()
			}
		}
	}
	sn = led.SN
	defer func() {
		beego.Info("led close sn:", sn)
		obj, ok := subscribers[sn]
		if ok {
			obj.Conn.Close()
			delete(subscribers, sn)
		} else {
			c.Close()
		}
	}()
}

// 发送命令给Led
func SendCommand(sn string, command string) string {
	led, ok := subscribers[sn]
	log.Println("xixun led switch", ok)
	if ok {
		led.Cond.L.Lock()
		log.Println("send command", command)
		led.Conn.WriteMessage(1, []byte(command))
		led.Cond.Wait()
		led.Cond.L.Unlock()
		beego.Info("led.Resp", led.Resp)
		beego.Info("led.Resp", &led.Resp)
		return led.Resp
	}
	return ""
}
