package websocketserver_test

import (
	"fmt"
	"testing"
	"time"

	_ "go-iot/pkg/codec"
	"go-iot/pkg/core"
	"go-iot/pkg/network"
	websocketserver "go-iot/pkg/network/servers/websocket"
	"go-iot/pkg/network/testhelper"
	_ "go-iot/pkg/timeseries"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

func startWS(t *testing.T, productId, script string) (port int32, cleanup func()) {
	t.Helper()
	port = testhelper.FreeTCPPort(t)
	testhelper.SetupProductDevice(t, productId, "dev-ws-ok")
	conf := network.NetworkConf{
		ProductId:     productId,
		CodecId:       "script_codec",
		Port:          port,
		Script:        script,
		Configuration: `{"host":"127.0.0.1","useTLS":false,"routers":[{"url":"/socket"}]}`,
	}
	srv := websocketserver.NewServer()
	require.NoError(t, srv.Start(conf))
	_, err := core.NewCodec(conf.CodecId, conf.ProductId, conf.Script)
	require.NoError(t, err)
	time.Sleep(50 * time.Millisecond)
	return port, func() { _ = srv.Stop() }
}

func TestWebSocket_ScriptError_DeviceOnlineUnknown(t *testing.T) {
	productId := fmt.Sprintf("ws-se-online-%d", time.Now().UnixNano()%100000)
	script := `
function OnConnect(context) {
  context.DeviceOnline(context.GetQuery("deviceId"))
}
function OnMessage(context) {
  context.GetSession().SendText(JSON.stringify({reached:true}))
}
`
	port, cleanup := startWS(t, productId, script)
	defer cleanup()

	u := fmt.Sprintf("ws://127.0.0.1:%d/socket?deviceId=ghost", port)
	conn, _, err := websocket.DefaultDialer.Dial(u, nil)
	require.NoError(t, err)
	defer conn.Close()
	time.Sleep(100 * time.Millisecond)
	require.Nil(t, core.GetSession("ghost"))

	// 正确设备仍可上线
	u2 := fmt.Sprintf("ws://127.0.0.1:%d/socket?deviceId=dev-ws-ok", port)
	conn2, _, err := websocket.DefaultDialer.Dial(u2, nil)
	require.NoError(t, err)
	defer conn2.Close()
	time.Sleep(100 * time.Millisecond)
	require.NotNil(t, core.GetSession("dev-ws-ok"))
}

func TestWebSocket_ScriptError_SavePropertiesEmptyDeviceId(t *testing.T) {
	productId := fmt.Sprintf("ws-se-prop-%d", time.Now().UnixNano()%100000)
	script := `
function OnConnect(context) {
  context.DeviceOnline(context.GetQuery("deviceId"))
}
function OnMessage(context) {
  // 故意不带 deviceId，且若 OnConnect 失败则 DeviceId 为空
  context.SaveProperties({temperature: 1})
  context.GetSession().SendText(JSON.stringify({reached:true}))
}
`
	port, cleanup := startWS(t, productId, script)
	defer cleanup()

	// 先用未知设备连接，DeviceOnline 失败，再发消息触发 SaveProperties 空 deviceId
	u := fmt.Sprintf("ws://127.0.0.1:%d/socket?deviceId=ghost", port)
	conn, _, err := websocket.DefaultDialer.Dial(u, nil)
	require.NoError(t, err)
	defer conn.Close()
	time.Sleep(50 * time.Millisecond)
	require.NoError(t, conn.WriteMessage(websocket.TextMessage, []byte(`{"temperature":1}`)))
	_ = conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	_, msg, err := conn.ReadMessage()
	// 脚本 throw 后不应收到 reached
	if err == nil {
		require.NotContains(t, string(msg), "reached")
	}
}

func TestWebSocket_ScriptError_ExplicitThrow(t *testing.T) {
	productId := fmt.Sprintf("ws-se-throw-%d", time.Now().UnixNano()%100000)
	script := `
function OnConnect(context) {
  context.DeviceOnline(context.GetQuery("deviceId"))
}
function OnMessage(context) {
  throw "ws boom"
}
`
	port, cleanup := startWS(t, productId, script)
	defer cleanup()
	u := fmt.Sprintf("ws://127.0.0.1:%d/socket?deviceId=dev-ws-ok", port)
	conn, _, err := websocket.DefaultDialer.Dial(u, nil)
	require.NoError(t, err)
	defer conn.Close()
	require.NoError(t, conn.WriteMessage(websocket.TextMessage, []byte(`{"deviceId":"dev-ws-ok","temperature":1}`)))
	time.Sleep(100 * time.Millisecond)
	// 连接与会话仍可存在，进程不崩
	require.NotNil(t, core.GetSession("dev-ws-ok"))
}
