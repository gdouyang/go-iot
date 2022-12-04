package mqttserver_test

import (
	"fmt"
	"go-iot/codec"
	"go-iot/codec/msg"
	"go-iot/models"
	_ "go-iot/models/device"
	mqttserver "go-iot/network/servers/mqtt"
	"os"
	"testing"
	"time"

	"github.com/beego/beego/v2/core/logs"
	MQTT "github.com/eclipse/paho.mqtt.golang"
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

var network codec.NetworkConf = codec.NetworkConf{
	Name:      "test server",
	ProductId: "test123",
	CodecId:   "script_codec",
	Port:      1883,
	Script:    script,
}

func TestServer(t *testing.T) {
	models.DefaultDbConfig.Url = "root:root@tcp(localhost:3306)/go-iot?charset=utf8&loc=Local&tls=false"
	models.InitDb()

	network := network
	network.Configuration = `{"host": "localhost", "useTLS": false}`
	b := mqttserver.NewServer()
	b.Start(network)
	go func() {
		time.Sleep(1 * time.Second)
		for i := 0; i < 5; i++ {
			go func() {
				err := codec.DoCmdInvoke(network.ProductId, msg.FuncInvoke{
					DeviceId:   "1234",
					FunctionId: "func1",
					Data:       map[string]interface{}{"name": "f"},
				})
				if err != nil {
					logs.Error(err)
				} else {
					logs.Info("cmdInvoke success")
				}
			}()
			time.Sleep(1 * time.Second)
		}
	}()

	newClient(network, "1234")
	// newClient(network, "4567")
}

func newClient(network codec.NetworkConf, deviceId string) {
	spec := mqttserver.MQTTServerSpec{}
	spec.FromJson(network.Configuration)
	spec.Port = network.Port
	opts := MQTT.NewClientOptions()
	opts.AddBroker("tcp://" + spec.Host + ":" + fmt.Sprint(spec.Port))
	opts.SetClientID(deviceId)
	opts.SetUsername("admin")
	opts.SetPassword("123456")
	opts.SetCleanSession(false)
	topic := "test"
	qos := 0
	payload := []byte(`{"temperature": 12.1, "fff":1}`)
	num := 5
	opts.SetDefaultPublishHandler(func(client MQTT.Client, msg MQTT.Message) {
		logs.Info("RECEIVED TOPIC: %s MESSAGE: %s", msg.Topic(), string(msg.Payload()))
		// reply cmd invoke
		go func() {
			client.Publish(topic, byte(qos), false, msg.Payload())
		}()
	})
	client := MQTT.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}
	if token := client.Subscribe(topic, byte(qos), nil); token.Wait() && token.Error() != nil {
		fmt.Println(token.Error())
		os.Exit(1)
	}
	fmt.Println("Sample Publisher Started")
	for i := 0; i < num; i++ {
		// fmt.Println("---- doing publish ----")
		token := client.Publish(topic, byte(qos), false, payload)
		token.Wait()
		time.Sleep(1 * time.Second)
	}

	client.Disconnect(250)
	fmt.Println("Sample Publisher Disconnected")
}
