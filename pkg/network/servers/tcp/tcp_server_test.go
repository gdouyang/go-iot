package tcpserver_test

import (
	"bufio"
	"fmt"
	_ "go-iot/pkg/codec"
	"go-iot/pkg/core"
	"go-iot/pkg/logger"
	"go-iot/pkg/network"
	tcpserver "go-iot/pkg/network/servers/tcp"
	"go-iot/pkg/store"
	_ "go-iot/pkg/timeseries"
	"net"
	"testing"
	"time"
)

const script = `
function OnConnect(context) {
  console.log("OnConnect: " + JSON.stringify(context))
}
function OnMessage(context) {
  console.log("OnMessage: " + context.MsgToString())
}
function OnInvoke(context) {
	console.log("OnInvoke: " + JSON.stringify(context))
}
function OnDeviceDeploy(context) {
	console.log(JSON.stringify(context))
}
function OnDeviceUnDeploy(context) {
	console.log(JSON.stringify(context))
}
function OnStateChecker(context) {
	console.log(JSON.stringify(context))
}
`

const script1 = `
function OnConnect(context) {
  console.log("OnConnect: " + JSON.stringify(context))
}
function OnMessage(context) {
	var data = JSON.parse(context.MsgToString())
  console.log("OnMessage: deviceId = " + data.deviceId)
	context.DeviceOnline(data.deviceId)
	context.SaveProperties({"msg": context.MsgToString()})
}
function OnInvoke(context) {
	console.log("OnInvoke: " + JSON.stringify(context))
}
`

var network1 network.NetworkConf = network.NetworkConf{
	Name:      "test server",
	ProductId: "test-product",
	CodecId:   "script_codec",
	Port:      8888,
	Script:    script,
}

func init() {
	logger.InitNop()
	core.RegDeviceStore(store.NewMockDeviceStore())
	var product *core.Product = &core.Product{
		Id:          "test-product",
		Config:      make(map[string]string),
		StorePolicy: "mock",
	}
	core.PutProduct(product)
	device := core.NewDevice("1234", product.Id, 0)
	core.PutDevice(device)
}

func newServer(network network.NetworkConf) *tcpserver.TcpServer {
	s := tcpserver.NewServer()
	s.Start(network)
	core.NewCodec(network.CodecId, network.ProductId, network.Script)
	return s
}
func TestServerDelimited(t *testing.T) {
	network := network1
	network.Configuration = `{"host": "localhost",
	"port": 8888, "useTLS": false,
	"delimeter": {"type":"Delimited", "delimited":"}"}}`
	network.Script = script1
	newServer(network)
	newClient1(network, func() string {
		str1 := time.Now().Format("2006-01-02 15:04:05")
		str := fmt.Sprintf(`{"deviceId": "1234", "data": "%s"}`, str1)
		return str
	})
}

func TestServerFixLenght(t *testing.T) {
	network := network1
	network.Configuration = `{"host": "localhost",
	"port": 8888, "useTLS": false,
	"delimeter": {"type":"FixLength", "length":27}}`
	newServer(network)
	newClient(network)
}

func TestServerSplitFunc(t *testing.T) {
	network := network1
	network.Configuration = `{"host": "localhost",
	"port": 8888, "useTLS": false,
	"delimeter": {
		"type":"SplitFunc",
	  "splitFunc":"function splitFunc(parser) { parser.AddHandler(function(data) { parser.AppendResult(data); parser.Complete() }); parser.Delimited(\"\\n\") }"
	}
	}`
	newServer(network)
	newClient(network)
}

func TestServerSplitFunc1(t *testing.T) {
	network := network1
	network.Configuration = `{"host": "localhost",
	"port": 8888, "useTLS": false,
	"delimeter": {
		"type":"SplitFunc",
	  "splitFunc":"function splitFunc(parser) { parser.AddHandler(function(data) { parser.AddHandler(function(data){ parser.AppendResult(data); parser.Complete() });  parser.Delimited(\"\\n\") }); parser.Delimited(\" \") }"
	}
	}`
	newServer(network)
	newClient(network)
}

func TestServerSplitFunc2(t *testing.T) {
	network := network1
	network.Configuration = `{"host": "localhost",
	"port": 8888, "useTLS": false,
	"delimeter": {
		"type":"SplitFunc",
	  "splitFunc":"function splitFunc(parser) { parser.AddHandler(function(data) { parser.AddHandler(function(data){ parser.AppendResult(data); parser.Complete() });  parser.Fixed(21) }); parser.Fixed(6) }"
	}
	}`
	newServer(network)
	newClient(network)
}

func newClient(network network.NetworkConf) {
	newClient1(network, func() string {
		str1 := time.Now().Format("2006-01-02 15:04:05")
		str := fmt.Sprintf("aasss %s_\n", str1)
		return str
	})
}

func newClient1(network network.NetworkConf, call func() string) {
	spec := tcpserver.TcpServerSpec{}
	spec.FromJson(network.Configuration)
	spec.Port = network.Port
	conn, err := net.Dial("tcp", spec.Host+":"+fmt.Sprint(spec.Port))
	if err != nil {
		fmt.Print(err)
	}
	go func() {
		stdin := bufio.NewScanner(conn)
		for stdin.Scan() {
			fmt.Println("server> " + stdin.Text())
		}
	}()

	for i := 0; i < 5; i++ {
		str := call()
		conn.Write([]byte(str))
		fmt.Println("client> " + str)

		time.Sleep(1 * time.Second)
	}
}
