package mqttclient

import (
	"testing"

	"go-iot/pkg/network"

	"github.com/stretchr/testify/require"
)

func TestMqttClientSpec_FromJson(t *testing.T) {
	spec := &MQTTClientSpec{}
	require.NoError(t, spec.FromJson(`{"host":"127.0.0.1","port":1883,"clientId":"c1","topics":{"a":0}}`))
	require.Equal(t, "c1", spec.ClientId)
	require.Equal(t, 0, spec.Topics["a"])
}

func TestMqttClient_Type(t *testing.T) {
	c := &MqttClient{}
	require.Equal(t, network.MQTT_CLIENT, c.Type())
}

func TestMqttClientSpec_InvalidJSON(t *testing.T) {
	spec := &MQTTClientSpec{}
	require.Error(t, spec.FromJson(`{`))
}
