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
	agent_server_ip = beego.AppConfig.String("agent_server_ip")
	agent_sn = beego.AppConfig.String("agent_sn")
	go conn()
}

var (
	agent_server_ip string
	agent_sn        string
)

// 连接iot中心管理端
func conn() {
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
	go heartbeat(conn)

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			fmt.Println("read:", err)
			return
		}

		fmt.Printf("received: %s\n", message)
	}
}

func heartbeat(conn *websocket.Conn) {
	for {
		ping := `{"sn":"` + agent_sn + `"}`
		conn.WriteMessage(websocket.TextMessage, []byte(ping))
		time.Sleep(time.Second * 20)
	}
}
