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

func TestServer(t *testing.T) {
	var network codec.Network = codec.Network{
		ProductId: "test",
		CodecId:   "script_codec",
		Script: `
function OnConnect(context) {
  console.log(JSON.stringify(context))
}
function Decode(context) {
  console.log("122")
  console.log(JSON.stringify(context))
}
function Encode(context) {
	console.log(JSON.stringify(context))
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
`,
	}
	tcpserver.ServerSocket(network)
	time.Sleep(10 * time.Second)
}

func newClient(network codec.Network) {
	spec := tcpserver.TcpServerSpec{}
	spec.FromJson(network.Configuration)
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

	for {
		str1 := time.Now().Format(time.RFC1123)
		str := fmt.Sprintf("aasss %s \n", str1)
		conn.Write([]byte(str))
		// fmt.Println("send ")

		time.Sleep(1 * time.Second)
	}
}
