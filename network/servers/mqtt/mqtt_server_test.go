package mqttserver_test

import (
	"fmt"
	"go-iot/codec"
	"go-iot/models"
	mqttserver "go-iot/network/servers/mqtt"
	"os"
	"testing"
	"time"

	MQTT "github.com/eclipse/paho.mqtt.golang"
)

func init() {
	models.DefaultDbConfig.Url = "root:root@tcp(localhost:3306)/go-iot?charset=utf8&loc=Local&tls=false"
	models.InitDb()
}

const script = `
function OnConnect(context) {
  console.log("OnConnect: " + context.GetClientId())
	context.DeviceOnline(context.GetClientId())
}
function OnMessage(context) {
  console.log("OnMessage: " + context.MsgToString())
  var data = JSON.parse(context.MsgToString())
  context.Save(data)
}
function OnInvoke(context) {
	console.log("OnInvoke: " + JSON.stringify(context))
}
function OnDeviceCreate(context) {
	console.log(JSON.stringify(context))
}
function OnDeviceDelete(context) {
	console.log(JSON.stringify(context))
}
function OnDeviceUpdate(context) {
	console.log(JSON.stringify(context))
}
function OnStateChecker(context) {
	console.log(JSON.stringify(context))
}
`

var network codec.Network = codec.Network{
	Name:      "test server",
	ProductId: "test123",
	CodecId:   "script_codec",
	Port:      1883,
	Script:    script,
}

func TestServer(t *testing.T) {
	network := network
	network.Configuration = `{"host": "localhost", "useTLS": false}`
	mqttserver.ServerStart(network)
	newClient(network)
}

func newClient(network codec.Network) {
	spec := mqttserver.MQTTServerSpec{}
	spec.FromJson(network.Configuration)
	spec.Port = network.Port
	opts := MQTT.NewClientOptions()
	opts.AddBroker("tcp://" + spec.Host + ":" + fmt.Sprint(spec.Port))
	opts.SetClientID("1234")
	opts.SetUsername("admin")
	opts.SetPassword("123456")
	opts.SetCleanSession(false)
	action := "pub"
	topic := "test"
	qos := 0
	payload := []byte(`{"temperature": 12.1}`)
	num := 10
	if action == "pub" {
		client := MQTT.NewClient(opts)
		if token := client.Connect(); token.Wait() && token.Error() != nil {
			panic(token.Error())
		}
		fmt.Println("Sample Publisher Started")
		for i := 0; i < num; i++ {
			fmt.Println("---- doing publish ----")
			token := client.Publish(topic, byte(qos), false, payload)
			token.Wait()
			time.Sleep(1 * time.Second)
		}

		client.Disconnect(250)
		fmt.Println("Sample Publisher Disconnected")
	} else {
		receiveCount := 0
		choke := make(chan [2]string)

		opts.SetDefaultPublishHandler(func(client MQTT.Client, msg MQTT.Message) {
			choke <- [2]string{msg.Topic(), string(msg.Payload())}
		})

		client := MQTT.NewClient(opts)
		if token := client.Connect(); token.Wait() && token.Error() != nil {
			panic(token.Error())
		}

		if token := client.Subscribe(topic, byte(qos), nil); token.Wait() && token.Error() != nil {
			fmt.Println(token.Error())
			os.Exit(1)
		}

		for receiveCount < num {
			incoming := <-choke
			fmt.Printf("RECEIVED TOPIC: %s MESSAGE: %s\n", incoming[0], incoming[1])
			receiveCount++
		}

		client.Disconnect(250)
		fmt.Println("Sample Subscriber Disconnected")
	}
}
