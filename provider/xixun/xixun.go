package xixun

import (
	"encoding/json"
	"fmt"
	"go-iot/models"
	"log"
	"net/http"

	"github.com/astaxie/beego"
	"github.com/gorilla/websocket"
)

var providerId string = "xixunled"

type ProviderXiXunLed struct {
	Id string //厂商ID
}

func (this ProviderXiXunLed) ProviderId() string {
	return this.Id
}

// 开关操作
func (this ProviderXiXunLed) Switch(status []models.SwitchStatus, device models.Device) {
	abc := "{\"type\": \"callCardService\",\"fn\": \"setScreenOpen\",\"arg1\": false}"
	led, ok := subscribers[device.Sn]
	log.Println("xixun led switch", ok)
	if ok {
		log.Println("send command", abc)
		led.Conn.WriteMessage(1, []byte(abc))
	}
}

type XixunLED struct {
	SN   string
	Conn *websocket.Conn
}

var (
	subscribers = map[string]XixunLED{}
)

type breath struct {
	Sn string `json:"cardId"`
}

func init() {
	port := beego.AppConfig.DefaultInt("xixunport", 7078)
	beego.Info(fmt.Sprintf("xixun init port:%d", port))
	var provider models.IProvider = ProviderXiXunLed{providerId}
	models.RegisterProvider(provider.ProviderId(), provider)
	go (func() {
		http.HandleFunc("/", upgradeWs)
		log.Fatal(http.ListenAndServe(fmt.Sprintf("%s:%d", "0.0.0.0", port), nil))
	})()
}

func upgradeWs(w http.ResponseWriter, r *http.Request) {
	c, err := websocket.Upgrade(w, r, nil, 1024, 1024)
	if err != nil {
		beego.Error("upgrade fail:", err)
		return
	}

	abc := breath{}
	var sn string
	for {
		mt, message, err := c.ReadMessage()
		json.Unmarshal(message, &abc)
		if err != nil {
			beego.Error("ws read err:", err)
			break
		}
		log.Println("message type:", mt)
		log.Println(abc.Sn)
		log.Println("message :", string(message))
		sn = abc.Sn
		_, ok := subscribers[sn]
		if !ok {
			subscribers[sn] = XixunLED{SN: sn, Conn: c}
			log.Println("led connection len", len(subscribers))
		}
	}
	if len(sn) > 0 {
		defer close(sn)
	}
}

func close(sn string) {
	obj := subscribers[sn]
	obj.Conn.Close()
	delete(subscribers, sn)
}
