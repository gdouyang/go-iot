package websocketsocker_test

import (
	"fmt"
	"go-iot/codec"
	"go-iot/models"
	_ "go-iot/models/device"
	websocketsocker "go-iot/network/servers/websocket"
	"log"
	"os"
	"os/signal"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

const script = `
function OnConnect(context) {
  console.log("OnConnect: ", context.GetQuery("deviceId"))
	context.DeviceOnline(context.GetQuery("deviceId"))
	console.log("DeviceOnline")
}
function OnMessage(context) {
	var msg = context.MsgToString()
  console.log("OnMessage: " + msg)
  var data = JSON.parse(msg)
  context.Save(data)
	context.GetSession().SendText(msg)
}
function OnInvoke(context) {
	console.log("OnInvoke: " + JSON.stringify(context))
}
`

var network codec.NetworkConf = codec.NetworkConf{
	Name:      "test server",
	ProductId: "test123",
	CodecId:   "script_codec",
	Port:      18080,
	Script:    script,
}

func TestServer(t *testing.T) {
	models.DefaultDbConfig.Url = "root:root@tcp(localhost:3306)/go-iot?charset=utf8&loc=Local&tls=false"
	models.InitDb()

	network := network
	network.Configuration = `{"host": "localhost", "useTLS": false, "paths":["/socket"]}`

	websocketsocker.NewServer().Start(network)

	c := &client{}
	go c.initClient("1234")
	c1 := &client{}
	c1.initClient("4567")
}

type client struct {
	done      chan interface{}
	interrupt chan os.Signal
}

func (c *client) receiveHandler(connection *websocket.Conn) {
	defer close(c.done)
	for {
		_, msg, err := connection.ReadMessage()
		if err != nil {
			log.Println("client Error in receive:", err)
			return
		}
		log.Printf("client Received: %s\n", msg)
	}
}

func (c *client) initClient(deviceId string) {
	c.done = make(chan interface{})    // Channel to indicate that the receiverHandler is done
	c.interrupt = make(chan os.Signal) // Channel to listen for interrupt signal to terminate gracefully

	signal.Notify(c.interrupt, os.Interrupt) // Notify the interrupt channel for SIGINT

	socketUrl := "ws://localhost:" + fmt.Sprint(network.Port) + "/socket?deviceId=" + deviceId
	conn, _, err := websocket.DefaultDialer.Dial(socketUrl, nil)
	if err != nil {
		log.Fatal("Error connecting to Websocket Server:", err)
	}
	defer conn.Close()
	go c.receiveHandler(conn)

	// Our main loop for the client
	// We send our relevant packets here
	count := 1
	for {
		select {
		case <-time.After(time.Duration(1) * time.Millisecond * 1000):
			// Send an echo packet every second
			err := conn.WriteMessage(websocket.TextMessage, []byte(`{"temperature": 16.1, "fff":1}`))
			if err != nil {
				log.Println("Error during writing to websocket:", err)
				return
			}
			count++
			if count == 10 {
				time.Sleep(time.Second * 2)
				return
			}

		case <-c.interrupt:
			// We received a SIGINT (Ctrl + C). Terminate gracefully...
			log.Println("Received SIGINT interrupt signal. Closing all pending connections")

			// Close our websocket connection
			err := conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Println("Error during closing websocket:", err)
				return
			}

			select {
			case <-c.done:
				log.Println("Receiver Channel Closed! Exiting....")
			case <-time.After(time.Duration(1) * time.Second):
				log.Println("Timeout in closing receiving channel. Exiting....")
			}
			return
		}
	}
}
