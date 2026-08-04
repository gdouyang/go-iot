package servers_test

import (
	"testing"

	_ "go-iot/pkg/codec"
	"go-iot/pkg/core"
	"go-iot/pkg/network"
	"go-iot/pkg/network/servers"
	_ "go-iot/pkg/network/servers/http"
	"go-iot/pkg/network/testhelper"
	_ "go-iot/pkg/timeseries"

	"github.com/stretchr/testify/require"
)

// 编解码创建阶段：非法脚本应失败，且不得注册为运行中的 server。
func TestStartServer_InvalidScriptRejected(t *testing.T) {
	port := testhelper.FreeTCPPort(t)
	productId := "p-bad-script"
	testhelper.InitCore()
	err := servers.StartServer(network.NetworkConf{
		ProductId:     productId,
		Type:          string(network.HTTP_SERVER),
		CodecId:       "script_codec",
		Port:          port,
		Script:        "function OnMessage( { invalid",
		Configuration: `{"host":"127.0.0.1"}`,
	})
	require.Error(t, err)
	require.Nil(t, servers.GetServer(productId))
}

func TestNewCodec_InvalidScript(t *testing.T) {
	testhelper.InitCore()
	_, err := core.NewCodec("script_codec", "p-codec-bad", "{{{{")
	require.Error(t, err)
}

func TestNewCodec_ValidButMissingOnMessage(t *testing.T) {
	testhelper.InitCore()
	// 可创建 codec；调用 OnMessage 时为 ErrFunctionNotImpl
	c, err := core.NewCodec("script_codec", "p-codec-miss", `function OnConnect(c){}`)
	require.NoError(t, err)
	require.NotNil(t, c)
	err = c.OnMessage(&core.BaseContext{ProductId: "p-codec-miss"})
	require.ErrorIs(t, err, core.ErrFunctionNotImpl)
}
