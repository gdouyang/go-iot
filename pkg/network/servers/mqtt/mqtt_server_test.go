package mqttserver_test

import (
	"fmt"
	_ "go-iot/pkg/codec"
	"go-iot/pkg/core"
	"go-iot/pkg/network"
	mqttserver "go-iot/pkg/network/servers/mqtt"
	"go-iot/pkg/store"
	_ "go-iot/pkg/timeseries"
	"go-iot/pkg/tsl"
	"os"
	"testing"
	"time"

	logs "go-iot/pkg/logger"

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
	opts := MQTT.NewClientOptions()
	opts.AddBroker("tcp://" + spec.Host + ":" + fmt.Sprint(spec.Port))
	opts.SetClientID(deviceId)
	// opts.SetUsername("admin")
	// opts.SetPassword("123456")
	opts.SetCleanSession(false)
	topic := "test"
	qos := 0
	payload := []byte(`{"temperature": 12.1, "fff":1}`)
	num := 5
	opts.SetDefaultPublishHandler(func(client MQTT.Client, msg MQTT.Message) {
		logs.Infof("RECEIVED TOPIC: %s MESSAGE: %s", msg.Topic(), string(msg.Payload()))
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
