package tcpserver_test

import (
	"bufio"
	"fmt"
	"go-iot/provider/codec"
	tcpserver "go-iot/provider/servers/tcp"
	"net"
	"testing"
	"time"
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
	Port:      8888,
	Script:    script,
}

func TestServerDelimited(t *testing.T) {
	network := network
	network.Configuration = `{"host": "localhost",
	"port": 8888, "useTLS": false,
	"delimeter": {"type":"Delimited", "delimited":"\n"}}`
	tcpserver.ServerSocket(network)
	newClient(network)
}

func TestServerFixLenght(t *testing.T) {
	network := network
	network.Configuration = `{"host": "localhost",
	"port": 8888, "useTLS": false,
	"delimeter": {"type":"FixLength", "length":27}}`
	tcpserver.ServerSocket(network)
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
	tcpserver.ServerSocket(network)
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
	tcpserver.ServerSocket(network)
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
	tcpserver.ServerSocket(network)
	newClient(network)
}

func newClient(network codec.Network) {
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
		str1 := time.Now().Format("2006-01-02 15:04:05")
		str := fmt.Sprintf("aasss %s \n", str1)
		conn.Write([]byte(str))

		time.Sleep(1 * time.Second)
	}
}
