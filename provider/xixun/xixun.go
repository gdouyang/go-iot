package xixun

import (
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
	//			abc := "{\"type\":\"callLiveService\",\"_type\":\"StartLiveVideo\",\"url\":\"rtsp://admin:admin@10.28.124.243:554/media/video3\",\"width\":168,\"height\":152}"
	//			abc := "{\"type\":\"callLiveService\",\"_type\":\"StartLiveVideo\",\"url\":\"rtmp://10.28.124.234:1935/live/abc\",\"width\":168,\"height\":152}"
	//			abc := "{\"type\":\"loadUrl\",\"url\":\"http://10.28.124.234:18070/index.html\",\"persistent\":true}"
	abc := "{\"type\":\"clear\"}"
	led, ok := subscribers[device.Sn]
	if ok {
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
	sn := r.RequestURI
	c, err := websocket.Upgrade(w, r, nil, 1024, 1024)
	if err != nil {
		beego.Error("upgrade fail:", err)
		return
	}
	subscribers[sn] = XixunLED{SN: sn, Conn: c}
	defer close(sn)
	for {
		mt, message, err := c.ReadMessage()
		if err != nil {
			beego.Error("ws read err:", err)
			break
		}
		log.Println("message type:", mt)
		log.Println("message :", string(message))
	}
}

func close(sn string) {
	obj := subscribers[sn]
	obj.Conn.Close()
	delete(subscribers, sn)
}
