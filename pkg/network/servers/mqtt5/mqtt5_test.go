package mqtt5_test

import (
	"context"
	"fmt"
	_ "go-iot/pkg/codec"
	"go-iot/pkg/core"
	"go-iot/pkg/network"
	mqttserver "go-iot/pkg/network/servers/mqtt5"
	"go-iot/pkg/store"
	_ "go-iot/pkg/timeseries"
	"go-iot/pkg/tsl"
	"net/url"
	"strconv"
	"testing"
	"time"

	logs "go-iot/pkg/logger"

	"github.com/eclipse/paho.golang/autopaho"
	"github.com/eclipse/paho.golang/paho"
)

const script = `
function OnConnect(context) {
  console.log("OnConnect: " + context.GetClientId())
	context.DeviceOnline(context.GetClientId())
}
function OnMessage(context) {
  console.log("OnMessage: " + context.MsgToString())
  var data = JSON.parse(context.MsgToString())
    if (data.name == 'f') {
		context.ReplyOk()
		return
	}
  context.SaveProperties(data)
}
function OnInvoke(context) {
	console.log("OnInvoke: " + JSON.stringify(context.GetMessage().Data))
	context.GetSession().Publish("test", JSON.stringify(context.GetMessage().Data))
}
`

var network1 network.NetworkConf = network.NetworkConf{
	Name:      "test server",
	ProductId: "test-product",
	CodecId:   "script_codec",
	Port:      1883,
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
	err := tslData.FromJson(`{"properties":[{"id":"temperature","type":"float"}],
	"functions":[{"id":"func1","inputs":[{"id":"name", "type":"string"}]}]}`)
	if err != nil {
		logs.Errorf(err.Error())
	}
	product.TslData = tslData
	core.PutProduct(product)
	device := core.NewDevice("1234", product.Id, 0)
	core.PutDevice(device)
}

func TestServer(t *testing.T) {
	network := network1
	network.Configuration = `{"host": "localhost", "useTLS": false}`
	b := mqttserver.NewServer()
	b.Start(network)
	core.NewCodec(network1.CodecId, network1.ProductId, network1.Script)
	go func() {
		time.Sleep(1 * time.Second)
		for i := 0; i < 5; i++ {
			go func() {
				err := core.DoCmdInvoke(core.FuncInvoke{
					DeviceId:   "1234",
					FunctionId: "func1",
					Data:       map[string]interface{}{"name": "f"},
				})
				if err != nil {
					logs.Errorf(err.Message)
				} else {
					logs.Infof("cmdInvoke success")
				}
			}()
			time.Sleep(1 * time.Second)
		}
	}()

	newClient(network, "1234")
	// newClient(network, "4567")
}

func newClient(network network.NetworkConf, deviceId string) {
	spec := mqttserver.MQTTServerSpec{}
	spec.FromJson(network.Configuration)
	spec.Port = network.Port
	// We will connect to the Eclipse test server (note that you may see messages that other users publish)
	u, err := url.Parse("mqtt://" + spec.Host + ":" + fmt.Sprint(spec.Port))
	if err != nil {
		panic(err)
	}
	topic := "test"
	ctx := context.Background()
	cliCfg := autopaho.ClientConfig{
		ServerUrls: []*url.URL{u},
		KeepAlive:  20, // Keepalive message should be sent every 20 seconds
		// CleanStartOnInitialConnection defaults to false. Setting this to true will clear the session on the first connection.
		CleanStartOnInitialConnection: false,
		// SessionExpiryInterval - Seconds that a session will survive after disconnection.
		// It is important to set this because otherwise, any queued messages will be lost if the connection drops and
		// the server will not queue messages while it is down. The specific setting will depend upon your needs
		// (60 = 1 minute, 3600 = 1 hour, 86400 = one day, 0xFFFFFFFE = 136 years, 0xFFFFFFFF = don't expire)
		SessionExpiryInterval: 60,
		OnConnectionUp: func(cm *autopaho.ConnectionManager, connAck *paho.Connack) {
			fmt.Println("mqtt connection up")
			// Subscribing in the OnConnectionUp callback is recommended (ensures the subscription is reestablished if
			// the connection drops)
			if _, err := cm.Subscribe(context.Background(), &paho.Subscribe{
				Subscriptions: []paho.SubscribeOptions{
					{Topic: topic, QoS: 1},
				},
			}); err != nil {
				fmt.Printf("failed to subscribe (%s). This is likely to mean no messages will be received.", err)
			}
			fmt.Println("mqtt subscription made")
		},
		OnConnectError: func(err error) { fmt.Printf("error whilst attempting connection: %s\n", err) },
		// eclipse/paho.golang/paho provides base mqtt functionality, the below config will be passed in for each connection
		ClientConfig: paho.ClientConfig{
			// If you are using QOS 1/2, then it's important to specify a client id (which must be unique)
			ClientID: deviceId,
			// OnPublishReceived is a slice of functions that will be called when a message is received.
			// You can write the function(s) yourself or use the supplied Router
			OnPublishReceived: []func(paho.PublishReceived) (bool, error){
				func(pr paho.PublishReceived) (bool, error) {
					fmt.Printf("received message on topic %s; body: %s (retain: %t)\n", pr.Packet.Topic, pr.Packet.Payload, pr.Packet.Retain)
					return true, nil
				}},
			OnClientError: func(err error) { fmt.Printf("client error: %s\n", err) },
			OnServerDisconnect: func(d *paho.Disconnect) {
				if d.Properties != nil {
					fmt.Printf("server requested disconnect: %s\n", d.Properties.ReasonString)
				} else {
					fmt.Printf("server requested disconnect; reason code: %d\n", d.ReasonCode)
				}
			},
		},
	}

	c, err := autopaho.NewConnection(ctx, cliCfg) // starts process; will reconnect until context cancelled
	if err != nil {
		panic(err)
	}
	// Wait for the connection to come up
	if err = c.AwaitConnection(ctx); err != nil {
		panic(err)
	}
	num := 5
	fmt.Println("Sample Publisher Started")
	for i := 0; i < num; i++ {
		// Publish a test message (use PublishViaQueue if you don't want to wait for a response)
		if _, err = c.Publish(ctx, &paho.Publish{
			QoS:     1,
			Topic:   topic,
			Payload: []byte("TestMessage: " + strconv.Itoa(i)),
		}); err != nil {
			if ctx.Err() == nil {
				panic(err) // Publish will exit when context cancelled or if something went wrong
			}
		}
	}
	fmt.Println("signal caught - exiting")
}
