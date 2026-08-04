package httpserver_test

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	_ "go-iot/pkg/codec"
	"go-iot/pkg/core"
	"go-iot/pkg/network"
	httpserver "go-iot/pkg/network/servers/http"
	"go-iot/pkg/network/testhelper"
	_ "go-iot/pkg/timeseries"

	"github.com/stretchr/testify/require"
)

// 脚本异常场景：DeviceOnline / SaveProperties / SaveEvents 返回 error 或 JS throw 时，
// 进程不应崩溃，HTTP 服务可继续处理后续请求。

func startHTTP(t *testing.T, productId, script string) (port int32, cleanup func()) {
	t.Helper()
	port = testhelper.FreeTCPPort(t)
	testhelper.SetupProductDevice(t, productId, "dev-ok")
	conf := network.NetworkConf{
		ProductId:     productId,
		CodecId:       "script_codec",
		Port:          port,
		Script:        script,
		Configuration: `{"host":"127.0.0.1","useTLS":false,"routers":[{"url":"/test"}]}`,
	}
	srv := httpserver.NewServer()
	require.NoError(t, srv.Start(conf))
	_, err := core.NewCodec(conf.CodecId, conf.ProductId, conf.Script)
	require.NoError(t, err)
	time.Sleep(50 * time.Millisecond)
	return port, func() { _ = srv.Stop() }
}

func postJSON(t *testing.T, url, body string) (int, string) {
	t.Helper()
	res, err := http.Post(url, "application/json", strings.NewReader(body))
	require.NoError(t, err)
	defer res.Body.Close()
	b, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	return res.StatusCode, string(b)
}

func TestHttp_ScriptError_DeviceOnlineUnknownDevice(t *testing.T) {
	productId := fmt.Sprintf("http-se-online-%d", time.Now().UnixNano()%100000)
	script := `
function OnMessage(context) {
  context.DeviceOnline(context.GetQuery("deviceId"))
  context.GetSession().ResponseJSON(JSON.stringify({reached:true}))
}
`
	port, cleanup := startHTTP(t, productId, script)
	defer cleanup()

	// 未知设备：DeviceOnline 返回 error → goja throw → 后续 Response 不会执行
	code, body := postJSON(t, fmt.Sprintf("http://127.0.0.1:%d/test?deviceId=ghost", port), `{}`)
	require.Equal(t, http.StatusOK, code) // 连接层仍 200
	require.NotContains(t, body, "reached")
	require.Nil(t, core.GetSession("ghost"))

	// 服务仍可用：正确设备可继续
	code, body = postJSON(t, fmt.Sprintf("http://127.0.0.1:%d/test?deviceId=dev-ok", port), `{}`)
	require.Equal(t, http.StatusOK, code)
	require.Contains(t, body, "reached")
}

func TestHttp_ScriptError_SavePropertiesEmptyDeviceId(t *testing.T) {
	productId := fmt.Sprintf("http-se-prop-%d", time.Now().UnixNano()%100000)
	script := `
function OnMessage(context) {
  // 不写 deviceId，且 context.DeviceId 为空
  context.SaveProperties({temperature: 1})
  context.GetSession().ResponseJSON(JSON.stringify({reached:true}))
}
`
	port, cleanup := startHTTP(t, productId, script)
	defer cleanup()

	code, body := postJSON(t, fmt.Sprintf("http://127.0.0.1:%d/test", port), `{"temperature":1}`)
	require.Equal(t, http.StatusOK, code)
	require.NotContains(t, body, "reached", "SaveProperties empty deviceId should throw and stop script")
}

func TestHttp_ScriptError_SaveEventsEmptyDeviceId(t *testing.T) {
	productId := fmt.Sprintf("http-se-evt-%d", time.Now().UnixNano()%100000)
	script := `
function OnMessage(context) {
  context.SaveEvents("alarm", {level: 1})
  context.GetSession().ResponseJSON(JSON.stringify({reached:true}))
}
`
	port, cleanup := startHTTP(t, productId, script)
	defer cleanup()

	code, body := postJSON(t, fmt.Sprintf("http://127.0.0.1:%d/test", port), `{}`)
	require.Equal(t, http.StatusOK, code)
	require.NotContains(t, body, "reached")
}

func TestHttp_ScriptError_ExplicitThrow(t *testing.T) {
	productId := fmt.Sprintf("http-se-throw-%d", time.Now().UnixNano()%100000)
	script := `
function OnMessage(context) {
  throw "manual script error"
  context.GetSession().ResponseJSON(JSON.stringify({reached:true}))
}
`
	port, cleanup := startHTTP(t, productId, script)
	defer cleanup()

	code, body := postJSON(t, fmt.Sprintf("http://127.0.0.1:%d/test", port), `{}`)
	require.Equal(t, http.StatusOK, code)
	require.NotContains(t, body, "reached")
}

func TestHttp_ScriptError_InvalidJSONInScript(t *testing.T) {
	productId := fmt.Sprintf("http-se-json-%d", time.Now().UnixNano()%100000)
	script := `
function OnMessage(context) {
  JSON.parse(context.MsgToString())
  context.GetSession().ResponseJSON(JSON.stringify({reached:true}))
}
`
	port, cleanup := startHTTP(t, productId, script)
	defer cleanup()

	code, body := postJSON(t, fmt.Sprintf("http://127.0.0.1:%d/test", port), `not-json`)
	require.Equal(t, http.StatusOK, code)
	require.NotContains(t, body, "reached")
}

func TestHttp_ScriptError_MissingOnMessage(t *testing.T) {
	productId := fmt.Sprintf("http-se-miss-%d", time.Now().UnixNano()%100000)
	// 无 OnMessage：FuncInvoke 返回 ErrFunctionNotImpl，不应崩溃
	script := `
function OnConnect(context) {}
`
	port, cleanup := startHTTP(t, productId, script)
	defer cleanup()

	code, _ := postJSON(t, fmt.Sprintf("http://127.0.0.1:%d/test", port), `{}`)
	require.Equal(t, http.StatusOK, code)
}

func TestHttp_ScriptError_ProductMismatchOnDeviceOnline(t *testing.T) {
	productId := fmt.Sprintf("http-se-prod-%d", time.Now().UnixNano()%100000)
	script := `
function OnMessage(context) {
  context.DeviceOnline(context.GetQuery("deviceId"))
  context.GetSession().ResponseJSON(JSON.stringify({reached:true}))
}
`
	port, cleanup := startHTTP(t, productId, script)
	defer cleanup()
	// 另一产品下的设备
	require.NoError(t, core.PutDevice(core.NewDevice("foreign", "other-product", 0)))

	code, body := postJSON(t, fmt.Sprintf("http://127.0.0.1:%d/test?deviceId=foreign", port), `{}`)
	require.Equal(t, http.StatusOK, code)
	require.NotContains(t, body, "reached")
}
