package clients

import (
	"errors"
	"go-iot/codec"
	mqttclient "go-iot/network/clients/mqtt"
	tcpclient "go-iot/network/clients/tcp"
)

func Connect(deviceId string, network codec.Network) error {
	switch network.Type {
	case codec.MQTT_CLIENT:
		mqttclient.ClientStart(deviceId, network)
		return nil
	case codec.TCP_CLIENT:
		tcpclient.ClientStart(deviceId, network)
		return nil
	}
	return errors.New("device is not client network")
}
