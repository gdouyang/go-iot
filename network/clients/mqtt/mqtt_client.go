package mqttclient

import (
	"go-iot/codec"
)

func ClientStart(deviceId string, network codec.Network) error {
	spec := MQTTClientSpec{}
	spec.FromJson(network.Configuration)
	spec.Port = network.Port

	session := newClientSession(deviceId, network, &spec)

	go session.readLoop()
	return nil
}
