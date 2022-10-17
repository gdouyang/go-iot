package mqttclient

import (
	"go-iot/codec"
)

type MqttClient struct {
	deviceId  string
	productId string
	spec      *MQTTClientSpec
	session   *clientSession
}

func (c *MqttClient) Type() codec.NetClientType {
	return codec.TCP_CLIENT
}

func (c *MqttClient) Start(deviceId string, network codec.NetworkConf) error {
	spec := MQTTClientSpec{}
	spec.FromJson(network.Configuration)
	spec.Port = network.Port

	session := newClientSession(deviceId, network, &spec)

	c.deviceId = deviceId
	c.productId = network.ProductId
	c.spec = &spec
	c.session = session

	go session.readLoop()

	return nil
}

func (c *MqttClient) Reload() error {
	return nil
}

func (c *MqttClient) Stop() error {
	return nil
}
