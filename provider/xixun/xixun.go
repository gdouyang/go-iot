package xixun

import (
	"fmt"
	"log"
	"net"
	"net/http"

	"github.com/astaxie/beego"
	ws "github.com/gorilla/websocket"
)

type XixunLED struct {
	SN   string
	Conn net.Conn
}

var AB = ws.Upgrader{}
var atre = true

func init() {
	port := beego.AppConfig.DefaultInt("xixunport", 7078)
	beego.Info(fmt.Sprintf("xixun init port:%d", port))
	go (func() {
		http.HandleFunc("/", process)
		log.Fatal(http.ListenAndServe(fmt.Sprintf("%s:%d", "0.0.0.0", port), nil))
	})()
}

func process(w http.ResponseWriter, r *http.Request) {
	c, err := AB.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	defer c.Close()
	for {
		mt, message, err := c.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			break
		}
		log.Println("message type:", mt)
		log.Println("message :", string(message))
		if atre {
			if len(message) > 0 {
				//			abc := "{\"type\":\"callLiveService\",\"_type\":\"StartLiveVideo\",\"url\":\"rtsp://admin:admin@10.28.124.243:554/media/video3\",\"width\":168,\"height\":152}"
				//			abc := "{\"type\":\"callLiveService\",\"_type\":\"StartLiveVideo\",\"url\":\"rtmp://10.28.124.234:1935/live/abc\",\"width\":168,\"height\":152}"
				//			abc := "{\"type\":\"loadUrl\",\"url\":\"http://10.28.124.234:18070/index.html\",\"persistent\":true}"
				abc := "{\"type\":\"clear\"}"
				c.WriteMessage(1, []byte(abc))
				atre = false
			}
		}
	}
}
