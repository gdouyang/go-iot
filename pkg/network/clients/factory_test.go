package clients_test

import (
	"testing"

	_ "go-iot/pkg/codec"
	"go-iot/pkg/network"
	"go-iot/pkg/network/clients"
	_ "go-iot/pkg/network/clients/modbus"
	_ "go-iot/pkg/network/clients/mqtt"
	_ "go-iot/pkg/network/clients/tcp"

	"github.com/stretchr/testify/require"
)

func TestConnectUnknownType(t *testing.T) {
	err := clients.Connect("d1", network.NetworkConf{
		Type:      "NOPE",
		CodecId:   "script_codec",
		ProductId: "p1",
		Script:    "function OnMessage(c){}",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknow client type")
}

func TestConnectRegisteredTypesNotUnknown(t *testing.T) {
	// 非法脚本在 codec 阶段失败
	types := []network.NetType{network.TCP_CLIENT, network.MQTT_CLIENT, network.MODBUS}
	for _, typ := range types {
		err := clients.Connect("d-"+string(typ), network.NetworkConf{
			Type:          string(typ),
			CodecId:       "script_codec",
			ProductId:     "p-" + string(typ),
			Script:        "{{{{not js",
			Configuration: `{"host":"127.0.0.1","port":1,"address":"127.0.0.1"}`,
		})
		require.Error(t, err, "type %s", typ)
		require.NotContains(t, err.Error(), "unknow client type", "type %s", typ)
	}
}

func TestGetClientMissing(t *testing.T) {
	require.Nil(t, clients.GetClient("no-client"))
}
