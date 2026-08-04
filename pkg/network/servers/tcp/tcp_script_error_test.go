package tcpserver_test

import (
	"fmt"
	"net"
	"testing"
	"time"

	_ "go-iot/pkg/codec"
	"go-iot/pkg/core"
	"go-iot/pkg/network"
	tcpserver "go-iot/pkg/network/servers/tcp"
	"go-iot/pkg/network/testhelper"
	_ "go-iot/pkg/timeseries"

	"github.com/stretchr/testify/require"
)

func startTCP(t *testing.T, productId, script string) (port int32, cleanup func()) {
	t.Helper()
	port = testhelper.FreeTCPPort(t)
	testhelper.SetupProductDevice(t, productId, "dev-tcp-ok")
	conf := network.NetworkConf{
		ProductId: productId,
		CodecId:   "script_codec",
		Port:      port,
		Script:    script,
		Configuration: fmt.Sprintf(`{"host":"127.0.0.1","port":%d,"useTLS":false,
			"delimeter":{"type":"Delimited","delimited":"}"}}`, port),
	}
	srv := tcpserver.NewServer()
	require.NoError(t, srv.Start(conf))
	_, err := core.NewCodec(conf.CodecId, conf.ProductId, conf.Script)
	require.NoError(t, err)
	time.Sleep(50 * time.Millisecond)
	return port, func() { _ = srv.Stop() }
}

func tcpSend(t *testing.T, port int32, payload string) {
	t.Helper()
	conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	require.NoError(t, err)
	defer conn.Close()
	_, err = conn.Write([]byte(payload))
	require.NoError(t, err)
	time.Sleep(150 * time.Millisecond)
}

func TestTcp_ScriptError_DeviceOnlineUnknown(t *testing.T) {
	productId := fmt.Sprintf("tcp-se-online-%d", time.Now().UnixNano()%100000)
	script := `
function OnMessage(context) {
  var data = JSON.parse(context.MsgToString())
  context.DeviceOnline(data.deviceId)
}
function OnConnect(context) {}
`
	port, cleanup := startTCP(t, productId, script)
	defer cleanup()

	tcpSend(t, port, `{"deviceId":"ghost"}`)
	require.Nil(t, core.GetSession("ghost"))

	// 仍可处理正确设备
	tcpSend(t, port, `{"deviceId":"dev-tcp-ok"}`)
	require.NotNil(t, core.GetSession("dev-tcp-ok"))
}

func TestTcp_ScriptError_SavePropertiesEmptyDeviceId(t *testing.T) {
	productId := fmt.Sprintf("tcp-se-prop-%d", time.Now().UnixNano()%100000)
	script := `
function OnMessage(context) {
  context.SaveProperties({temperature: 1})
}
function OnConnect(context) {}
`
	port, cleanup := startTCP(t, productId, script)
	defer cleanup()
	// 不应导致服务崩溃；再次连接仍成功
	tcpSend(t, port, `{"x":1}`)
	conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	require.NoError(t, err)
	conn.Close()
}

func TestTcp_ScriptError_ExplicitThrow(t *testing.T) {
	productId := fmt.Sprintf("tcp-se-throw-%d", time.Now().UnixNano()%100000)
	script := `
function OnMessage(context) {
  throw "tcp script boom"
}
function OnConnect(context) {}
`
	port, cleanup := startTCP(t, productId, script)
	defer cleanup()
	tcpSend(t, port, `{"deviceId":"dev-tcp-ok"}`)
	// 服务仍在监听
	conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	require.NoError(t, err)
	conn.Close()
}
