package tcpserver_test

import (
	"bufio"
	"fmt"
	"go-iot/codec"
	"go-iot/models"
	_ "go-iot/models/device"
	tcpserver "go-iot/network/servers/tcp"
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

var network codec.NetworkConf = codec.NetworkConf{
	Name:      "test server",
	ProductId: "test-product",
	CodecId:   "script_codec",
	Port:      8888,
	Script:    script,
}

var product *codec.Product = &codec.Product{
	Id:          "test-product",
	Config:      make(map[string]string),
	StorePolicy: "mock",
}

func init() {
	codec.DefaultManagerId = "mem"
	models.DefaultDbConfig.Url = "root:root@tcp(localhost:3306)/go-iot?charset=utf8&loc=Local&tls=false"
	models.InitDb()
	codec.PutProduct(product)
	device := &codec.Device{
		Id:        "1234",
		ProductId: product.GetId(),
		Data:      make(map[string]string),
		Config:    make(map[string]string),
	}
	codec.PutDevice(device)
}

func newServer(network codec.NetworkConf) *tcpserver.TcpServer {
	s := tcpserver.NewServer()
	s.Start(network)
	codec.NewCodec(network)
	return s
}
func TestServerDelimited(t *testing.T) {
	network := network
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
	network := network
	network.Configuration = `{"host": "localhost",
	"port": 8888, "useTLS": false,
	"delimeter": {"type":"FixLength", "length":27}}`
	newServer(network)
	newClient(network)
}

func TestServerSplitFunc(t *testing.T) {
	network := network
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
	network := network
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
	network := network
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

func newClient(network codec.NetworkConf) {
	newClient1(network, func() string {
		str1 := time.Now().Format("2006-01-02 15:04:05")
		str := fmt.Sprintf("aasss %s_\n", str1)
		return str
	})
}

func newClient1(network codec.NetworkConf, call func() string) {
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

	for i := 0; i < 10; i++ {
		str := call()
		conn.Write([]byte(str))

		time.Sleep(1 * time.Second)
	}
}
