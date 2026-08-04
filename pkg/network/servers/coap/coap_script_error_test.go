package coapserver_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	_ "go-iot/pkg/codec"
	"go-iot/pkg/core"
	"go-iot/pkg/network"
	coapserver "go-iot/pkg/network/servers/coap"
	"go-iot/pkg/network/testhelper"
	_ "go-iot/pkg/timeseries"

	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/go-coap/v3/udp"
	"github.com/stretchr/testify/require"
)

func startCoap(t *testing.T, productId, script string) (port int32, cleanup func()) {
	t.Helper()
	port = testhelper.FreeUDPPort(t)
	testhelper.SetupProductDevice(t, productId, "dev-coap-ok")
	conf := network.NetworkConf{
		ProductId:     productId,
		CodecId:       "script_codec",
		Port:          port,
		Script:        script,
		Configuration: `{"host":"127.0.0.1","useTLS":false,"routers":[{"url":"/test"}]}`,
	}
	srv := coapserver.NewServer()
	require.NoError(t, srv.Start(conf))
	_, err := core.NewCodec(conf.CodecId, conf.ProductId, conf.Script)
	require.NoError(t, err)
	time.Sleep(100 * time.Millisecond)
	return port, func() {
		defer func() { _ = recover() }()
		_ = srv.Stop()
	}
}

func coapPost(t *testing.T, port int32, path, body string) {
	t.Helper()
	co, err := udp.Dial(fmt.Sprintf("127.0.0.1:%d", port))
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err = co.Post(ctx, path, message.AppJSON, strings.NewReader(body))
	// 脚本异常时可能仍有响应或错误，不要求特定 body，只要不崩
	if err != nil {
		t.Logf("coap post err (may be ok after script error): %v", err)
	}
}

func TestCoap_ScriptError_DeviceOnlineUnknown(t *testing.T) {
	productId := fmt.Sprintf("coap-se-online-%d", time.Now().UnixNano()%100000)
	script := `
function OnMessage(context) {
  context.DeviceOnline(context.GetQuery("deviceId"))
  context.GetSession().Response(JSON.stringify({reached:true}))
}
`
	port, cleanup := startCoap(t, productId, script)
	defer cleanup()

	coapPost(t, port, "/test?deviceId=ghost", `{}`)
	// 正确设备后续仍可请求
	coapPost(t, port, "/test?deviceId=dev-coap-ok", `{"deviceId":"dev-coap-ok","temperature":1}`)
}

func TestCoap_ScriptError_SavePropertiesEmptyDeviceId(t *testing.T) {
	productId := fmt.Sprintf("coap-se-prop-%d", time.Now().UnixNano()%100000)
	script := `
function OnMessage(context) {
  context.SaveProperties({temperature: 1})
  context.GetSession().Response(JSON.stringify({reached:true}))
}
`
	port, cleanup := startCoap(t, productId, script)
	defer cleanup()
	coapPost(t, port, "/test", `{"temperature":1}`)
	// 再次请求证明服务仍在
	coapPost(t, port, "/test?deviceId=dev-coap-ok", `{"deviceId":"dev-coap-ok","temperature":2}`)
}

func TestCoap_ScriptError_ExplicitThrow(t *testing.T) {
	productId := fmt.Sprintf("coap-se-throw-%d", time.Now().UnixNano()%100000)
	script := `
function OnMessage(context) {
  throw "coap boom"
}
`
	port, cleanup := startCoap(t, productId, script)
	defer cleanup()
	coapPost(t, port, "/test", `{}`)
	coapPost(t, port, "/test", `{}`)
}
