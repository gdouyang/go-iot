package network_test

import (
	"testing"

	"go-iot/pkg/network"

	"github.com/stretchr/testify/assert"
)

func TestNetTypeConstants(t *testing.T) {
	servers := []network.NetType{
		network.MQTT_BROKER,
		network.GOIOT_MQTT_BROKER,
		network.TCP_SERVER,
		network.HTTP_SERVER,
		network.WEBSOCKET_SERVER,
		network.COAP_SERVER,
	}
	clients := []network.NetType{
		network.MQTT_CLIENT,
		network.TCP_CLIENT,
		network.MODBUS,
	}
	for _, s := range servers {
		assert.False(t, network.IsNetClientType(string(s)), "server type %s should not be client", s)
	}
	for _, c := range clients {
		assert.True(t, network.IsNetClientType(string(c)), "client type %s", c)
	}
}

func TestIsValidNetType(t *testing.T) {
	for _, typ := range []network.NetType{
		network.MQTT_BROKER,
		network.GOIOT_MQTT_BROKER,
		network.TCP_SERVER,
		network.HTTP_SERVER,
		network.WEBSOCKET_SERVER,
		network.COAP_SERVER,
		network.MQTT_CLIENT,
		network.TCP_CLIENT,
		network.MODBUS,
	} {
		assert.True(t, network.IsValidNetType(string(typ)), typ)
	}
	assert.False(t, network.IsValidNetType(""))
	assert.False(t, network.IsValidNetType("MQTT"))
	assert.False(t, network.IsValidNetType("mqtt"))
	assert.False(t, network.IsValidNetType("TCP"))
}

func TestIsStateless(t *testing.T) {
	assert.True(t, network.IsStateless(string(network.HTTP_SERVER)))
	assert.False(t, network.IsStateless(string(network.TCP_SERVER)))
	assert.False(t, network.IsStateless(string(network.MQTT_BROKER)))
	assert.False(t, network.IsStateless(string(network.WEBSOCKET_SERVER)))
}

func TestNetworkMetaConfigEmpty(t *testing.T) {
	cfg := network.GetNetworkMetaConfig("not-exist-type")
	assert.Empty(t, cfg.MetaConfigs)
}
