package websocketserver_test

import (
	"fmt"
	_ "go-iot/pkg/codec"
	"go-iot/pkg/core"
	"go-iot/pkg/network"
	websocketserver "go-iot/pkg/network/servers/websocket"
	"go-iot/pkg/store"
	_ "go-iot/pkg/timeseries"
	"go-iot/pkg/tsl"
	"log"
	"os"
	"os/signal"
	"testing"
	"time"

	logs "go-iot/pkg/logger"

	"github.com/gorilla/websocket"
)

const script = `
function OnConnect(context) {
	var deviceId = context.GetQuery("deviceId")
  console.log("OnConnect: " + deviceId)
	context.DeviceOnline(deviceId)
	console.log("DeviceOnline:" + deviceId)
}
function OnMessage(context) {
	var msg = context.MsgToString()
  console.log("OnMessage: " + msg)
  var data = JSON.parse(msg)
  context.SaveProperties(data)
	context.GetSession().SendText(msg)
}
function OnInvoke(context) {
	console.log("OnInvoke: " + JSON.stringify(context))
}
`

var network1 network.NetworkConf = network.NetworkConf{
	Name:      "test server",
	ProductId: "test-product",
	CodecId:   "script_codec",
	Port:      18080,
	Script:    script,
}

func init() {
	logs.InitNop()
	core.RegDeviceStore(store.NewMockDeviceStore())
	var product *core.Product = &core.Product{
		Id:          "test-product",
		Config:      make(map[string]string),
		StorePolicy: "mock",
	}
	tslData := &tsl.TslData{}
	err := tslData.FromJson(`{"properties":[{"id":"temperature","type":"float"}],"functions":[{"id":"func1","inputs":[{"id":"name", "type":"string"}]}]}`)
	if err != nil {
		logs.Errorf(err.Error())
	}
	product.TslData = tslData
	core.PutProduct(product)
	{
		device := core.NewDevice("1234", product.Id, 0)
		core.PutDevice(device)
	}
	{
		device := core.NewDevice("4567", product.Id, 0)
		core.PutDevice(device)
	}
}

func TestServer(t *testing.T) {
	network := network1
	network.Configuration = `{"host": "localhost", "useTLS": false, "paths":["/socket"]}`

	websocketserver.NewServer().Start(network)
	core.NewCodec(network1.CodecId, network1.ProductId, network1.Script)

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

	socketUrl := "ws://localhost:" + fmt.Sprint(network1.Port) + "/socket?deviceId=" + deviceId
	conn, _, err := websocket.DefaultDialer.Dial(socketUrl, nil)
	if err != nil {
		panic(fmt.Errorf("Error connecting to Websocket Server: %v", err))
	}
	defer conn.Close()
	go c.receiveHandler(conn)

	// Our main loop for the client
	// We send our relevant packets here
	count := 1
	ticker := time.NewTicker(time.Second * 1)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
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
			case <-ticker.C:
				log.Println("Timeout in closing receiving channel. Exiting....")
			}
			return
		}
	}
}
