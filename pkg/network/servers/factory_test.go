package servers_test

import (
	"fmt"
	"testing"

	_ "go-iot/pkg/codec"
	"go-iot/pkg/network"
	"go-iot/pkg/network/servers"
	_ "go-iot/pkg/network/servers/coap"
	_ "go-iot/pkg/network/servers/http"
	_ "go-iot/pkg/network/servers/mqtt"
	_ "go-iot/pkg/network/servers/mqtt5"
	_ "go-iot/pkg/network/servers/tcp"
	_ "go-iot/pkg/network/servers/websocket"

	"github.com/stretchr/testify/require"
)

func TestStartServerUnknownType(t *testing.T) {
	err := servers.StartServer(network.NetworkConf{
		ProductId: "p-unknown",
		Type:      "NOT_A_REAL_TYPE",
		CodecId:   "script_codec",
		Port:      1,
		Script:    "function OnMessage(c){}",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknow type")
}

func TestStopServerNotRunning(t *testing.T) {
	err := servers.StopServer("never-started")
	require.Error(t, err)
}

func TestStartServerRegisteredTypesNotUnknown(t *testing.T) {
	// 已注册类型：非法脚本应在 codec 阶段失败，而不是 unknow type
	// 注意：mqtt 与 mqtt5 的 Type() 均为 MQTT_BROKER，后注册者覆盖前者（现有实现）
	types := []network.NetType{
		network.HTTP_SERVER,
		network.TCP_SERVER,
		network.WEBSOCKET_SERVER,
		network.COAP_SERVER,
		network.MQTT_BROKER,
	}
	for i, typ := range types {
		err := servers.StartServer(network.NetworkConf{
			ProductId:     fmt.Sprintf("p-reg-%s-%d", typ, i),
			Type:          string(typ),
			CodecId:       "script_codec",
			Port:          int32(19000 + i),
			Script:        "this is not valid js{{{{",
			Configuration: `{"host":"127.0.0.1"}`,
		})
		require.Error(t, err, "type %s", typ)
		require.NotContains(t, err.Error(), "unknow type", "type %s should be registered", typ)
	}
}
