package mqtt5

import (
	"testing"

	"go-iot/pkg/network"

	"github.com/stretchr/testify/require"
)

func TestMQTTServerSpec_FromJson(t *testing.T) {
	spec := &MQTTServerSpec{}
	require.NoError(t, spec.FromJson(`{"host":"127.0.0.1","port":1883,"name":"b1","useTLS":false}`))
	require.Equal(t, "127.0.0.1", spec.Host)
	require.Equal(t, int32(1883), spec.Port)
	require.Equal(t, "b1", spec.Name)
	require.False(t, spec.UseTLS)
}

func TestMQTTServerSpec_FromJsonEmpty(t *testing.T) {
	spec := &MQTTServerSpec{}
	require.NoError(t, spec.FromJson(""))
}

func TestMQTTServerSpec_FromJsonInvalid(t *testing.T) {
	spec := &MQTTServerSpec{}
	require.Error(t, spec.FromJson(`{bad`))
}

func TestMQTTServerSpec_FromNetwork(t *testing.T) {
	spec := &MQTTServerSpec{}
	require.NoError(t, spec.FromNetwork(network.NetworkConf{
		Configuration: `{"host":"0.0.0.0","port":9}`,
	}))
	require.Equal(t, "0.0.0.0", spec.Host)
}

func TestBroker_Type(t *testing.T) {
	require.Equal(t, network.MQTT_BROKER, NewServer().Type())
}
