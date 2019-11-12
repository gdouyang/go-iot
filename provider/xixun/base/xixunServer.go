// XiXun服务器提供WebSocket服务控制设备
package base

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"regexp"
	"strings"

	"go-iot/models"
	"go-iot/models/modelfactory"
	"go-iot/models/operates"
	"go-iot/provider/util"
	"net/http"
	"sync"
	"time"

	"github.com/astaxie/beego"
	"github.com/gorilla/websocket"
)

type XixunLED struct {
	uid      int             //WebSocket连接唯一标识
	SN       string          //设备SN
	Conn     *websocket.Conn // websocket连接
	Cond     *sync.Cond      // 同步调用的condition
	respChan chan int        // 命令响应Channel
	Resp     string          // 命令返回
}

// 心跳
type breath struct {
	Sn string `json:"cardId"`
}

var (
	subscribers = map[string]*XixunLED{}
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
	led := &XixunLED{uid: util.Uuid(), SN: sn, Conn: c, Cond: sync.NewCond(&l), respChan: make(chan int, 2)}
	for {
		mt, message, err := c.ReadMessage()
		if err != nil {
			beego.Error("ws read err:", err)
			break
		}
		resp := string(message)
		abc := breath{}
		json.Unmarshal(message, &abc)
		sn = abc.Sn
		if len(sn) > 0 {
			beego.Info("breath -----> ", sn, "mssageType:", mt, "message :", resp)
			_, ok := subscribers[sn]
			if !ok {
				led.SN = sn
				subscribers[sn] = led
				evt := operates.DeviceOnlineStatus{OnlineStatus: models.ONLINE, Sn: sn, Provider: providerId}
				modelfactory.FireOnlineStatus(evt)
				beego.Info("led connected, connection len:", len(subscribers))
			}
		} else {
			l, ok := subscribers[led.SN]
			if ok {
				beego.Info("response -----> ", led.SN, "mssageType:", mt, "message :", resp)
				l.Cond.L.Lock()
				l.Resp = resp // 返回响应消息
				l.Cond.Signal()
				l.Cond.L.Unlock()
				l.respChan <- 1
			} else {
				beego.Warn("not found connection sn:", led.SN)
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
		evt := operates.DeviceOnlineStatus{OnlineStatus: models.OFFLINE, Sn: sn, Provider: providerId}
		modelfactory.FireOnlineStatus(evt)
	}()
}

// 获取在线状态
func GetOnlineStatus(sn string) string {
	_, ok := subscribers[sn]
	if ok {
		return models.ONLINE
	}
	return models.OFFLINE
}

// 发送命令给Led，等待Led给出响应后返回
func SendCommand(sn string, command string) (string, error) {
	led, ok := subscribers[sn]
	if ok {
		// LED没有返回的情况需要处理超时
		go checkTimeout(led)
		// 把当前请求等待,等待接口返回
		led.Cond.L.Lock()
		beego.Info("send command", command)
		led.Conn.WriteMessage(1, []byte(command))
		led.Cond.Wait()
		led.Cond.L.Unlock()
		beego.Info("led.Resp", led.Resp)
		return led.Resp, nil
	}
	return "", errors.New(sn + "没有在线")
}

// LED没有返回的情况需要处理超时
func checkTimeout(led *XixunLED) {
	select {
	case <-led.respChan:
		beego.Info("send command success resp")
	case <-time.Tick(time.Second * 20):
		led.Cond.L.Lock()
		beego.Info("send command has timeout")
		led.Resp = "timeout"
		led.Cond.Signal()
		led.Cond.L.Unlock()
	}
}

// 设备自动发现 等待秒数，返回json串
func DeviceDispear(duration time.Duration) string {
	liste := map[string]string{}
	connLister, err := net.ResolveUDPAddr("udp", "0.0.0.0:31306")
	if err != nil {
		fmt.Print(err)
		return ""
	}
	conn1, err := net.ListenUDP("udp", connLister)
	if err != nil {
		fmt.Print(err)
		return ""
	}
	defer conn1.Close()
	go func() {
		buf1 := make([]byte, 1024)
		for {
			n, p, err := conn1.ReadFromUDP(buf1)
			if err != nil {
				fmt.Print(err)
				continue
			}

			reg := regexp.MustCompile(`DisplayID=[a-z A-Z 0-9 \-]*`)
			cardIds := reg.FindAllString(string(buf1[:n]), -1)[0]
			cardId := strings.Split(cardIds, "=")
			//			fmt.Println(cardId)
			//			fmt.Println(p.IP.String())
			liste[cardId[1]] = p.IP.String()
		}
	}()
	// 时间到了就退出
	time.Sleep(duration * time.Second)
	jsonStr, err := json.Marshal(liste)
	if err != nil {
		fmt.Println("MapToJsonDemo err: ", err)
		return ""
	}
	return string(jsonStr)

}
