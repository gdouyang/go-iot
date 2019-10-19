package agent

import (
	"fmt"
	"net/url"
	"time"

	"github.com/astaxie/beego"
	"github.com/gorilla/websocket"
)

func init() {
	agentmode := beego.AppConfig.String("agentmode")
	if agentmode != "true" {
		return
	}
	go conn()
}

// 连接iot中心管理端
func conn() {
	agent_server_ip := beego.AppConfig.String("agent_server_ip")
	if len(agent_server_ip) == 0 {
		beego.Error("agent_server_ip len is 0")
	}
	time.Sleep(time.Second * 10)
	u := url.URL{Scheme: "ws", Host: agent_server_ip, Path: "/agetn/ws"}

	var dialer *websocket.Dialer

	conn, _, err := dialer.Dial(u.String(), nil)
	if err != nil {
		fmt.Println(err)
		return
	}
	beego.Info("iot center server connected")
	go timeWriter(conn)

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			fmt.Println("read:", err)
			return
		}

		fmt.Printf("received: %s\n", message)
	}
}

func timeWriter(conn *websocket.Conn) {
	for {
		time.Sleep(time.Second * 20)
		ping := `{"sn":"agent123456"}`
		conn.WriteMessage(websocket.TextMessage, []byte(ping))
	}
}
