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
	"go-iot/pkg/timeseries"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

const script = `
function OnConnect(context) {
	var deviceId = context.GetQuery("deviceId")
	context.DeviceOnline(deviceId)
}
function OnMessage(context) {
	var data = JSON.parse(context.MsgToString())
	context.SaveProperties(data)
	context.GetSession().SendText(JSON.stringify({ok:true}))
}
function OnInvoke(context) {}
`

func TestWebSocketServer_OnlineAndProperties(t *testing.T) {
	timeseries.MockReset()
	port := testhelper.FreeTCPPort(t)
	productId := fmt.Sprintf("ws-p-%d", port)
	deviceId := "dev-ws-1"
	testhelper.SetupProductDevice(t, productId, deviceId)

	conf := network.NetworkConf{
		Name:          "ws-it",
		ProductId:     productId,
		CodecId:       "script_codec",
		Port:          port,
		Script:        script,
		Configuration: `{"host":"127.0.0.1","useTLS":false,"paths":["/socket"]}`,
	}
	// paths 可能不在 spec 里，用 routers
	conf.Configuration = `{"host":"127.0.0.1","useTLS":false,"routers":[{"url":"/socket"}]}`

	srv := websocketserver.NewServer()
	require.NoError(t, srv.Start(conf))
	t.Cleanup(func() { _ = srv.Stop() })
	_, err := core.NewCodec(conf.CodecId, conf.ProductId, conf.Script)
	require.NoError(t, err)
	time.Sleep(50 * time.Millisecond)

	u := fmt.Sprintf("ws://127.0.0.1:%d/socket?deviceId=%s", port, deviceId)
	conn, _, err := websocket.DefaultDialer.Dial(u, nil)
	require.NoError(t, err)
	defer conn.Close()

	time.Sleep(100 * time.Millisecond)
	require.NotNil(t, core.GetSession(deviceId), "device should be online after ws connect")

	before := timeseries.MockPropertiesCount()
	msg := fmt.Sprintf(`{"deviceId":"%s","temperature":16.1}`, deviceId)
	require.NoError(t, conn.WriteMessage(websocket.TextMessage, []byte(msg)))

	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, resp, err := conn.ReadMessage()
	require.NoError(t, err)
	require.Contains(t, string(resp), "ok")

	require.Greater(t, timeseries.MockPropertiesCount(), before)
	saved := timeseries.MockLastProperties()
	require.NotNil(t, saved)
	require.Equal(t, deviceId, fmt.Sprint(saved["deviceId"]))
	require.EqualValues(t, 16.1, saved["temperature"])
}

func TestWebSocketServer_UnknownDeviceOnlineError(t *testing.T) {
	port := testhelper.FreeTCPPort(t)
	productId := fmt.Sprintf("ws-p2-%d", port)
	testhelper.SetupProductDevice(t, productId, "real-dev")

	conf := network.NetworkConf{
		ProductId:     productId,
		CodecId:       "script_codec",
		Port:          port,
		Script:        script,
		Configuration: `{"host":"127.0.0.1","useTLS":false,"routers":[{"url":"/socket"}]}`,
	}
	srv := websocketserver.NewServer()
	require.NoError(t, srv.Start(conf))
	t.Cleanup(func() { _ = srv.Stop() })
	_, err := core.NewCodec(conf.CodecId, conf.ProductId, conf.Script)
	require.NoError(t, err)
	time.Sleep(50 * time.Millisecond)

	u := fmt.Sprintf("ws://127.0.0.1:%d/socket?deviceId=ghost", port)
	conn, _, err := websocket.DefaultDialer.Dial(u, nil)
	require.NoError(t, err)
	defer conn.Close()
	time.Sleep(100 * time.Millisecond)
	// DeviceOnline 失败：不应挂上 ghost session
	require.Nil(t, core.GetSession("ghost"))
}
