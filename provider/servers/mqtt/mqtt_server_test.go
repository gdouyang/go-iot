package mqttserver_test

import (
	"fmt"
	"go-iot/provider/codec"
	mqttserver "go-iot/provider/servers/mqtt"
	"log"
	"net"
	"testing"
	"time"

	"github.com/eclipse/paho.mqtt.golang/packets"
)

const script = `
function OnConnect(context) {
  console.log("OnConnect: " + JSON.stringify(context))
}
function Decode(context) {
  console.log("Decode: " + context.MsgToString())
}
function Encode(context) {
	console.log("Encode: " + JSON.stringify(context))
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
	ProductId: "test",
	CodecId:   "script_codec",
	Port:      8889,
	Script:    script,
}

func TestServerDelimited(t *testing.T) {
	network := network
	network.Configuration = `{"host": "localhost",
	"port": 8888, "useTLS": false}`
	mqttserver.ServerStart(network)
	newClient(network)
}

func newClient(network codec.Network) {
	spec := mqttserver.MQTTServerSpec{}
	spec.FromJson(network.Configuration)
	spec.Port = network.Port
	conn, err := net.Dial("tcp", spec.Host+":"+fmt.Sprint(spec.Port))
	if err != nil {
		fmt.Print(err)
	}
	go func() {
		for {
			packet, err := packets.ReadPacket(conn)
			if err != nil {
				log.Printf("read packet failed: %v", err)
				continue
			}
			fmt.Println("server> " + packet.String())
		}
	}()
	// s := packets.NewControlPacket(packets.Subscribe).(*packets.SubscribePacket)
	// s.Topics = []string{"test"}
	// s.Write(conn)

	for i := 0; i < 10; i++ {
		str1 := time.Now().Format("2006-01-02 15:04:05")
		str := fmt.Sprintf("aasss %s \n", str1)
		p := packets.NewControlPacket(packets.Publish).(*packets.PublishPacket)
		p.Payload = []byte(str)
		p.Qos = 0
		p.TopicName = "test"
		p.Write(conn)

		time.Sleep(1 * time.Second)
	}
}
