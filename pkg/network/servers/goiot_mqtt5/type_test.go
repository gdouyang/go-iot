package goiot_mqtt5

import (
	"testing"

	"go-iot/pkg/network"

	"github.com/stretchr/testify/require"
)

func TestGoiotMqtt5_Type(t *testing.T) {
	b := &Broker{}
	require.Equal(t, network.GOIOT_MQTT_BROKER, b.Type())
}
