package mqttclient

import (
	"errors"
	"go-iot/codec"
	"go-iot/network/clients"
)

func init() {
	clients.RegClient(func() codec.NetClient {
		return &MqttClient{}
	})
}

type MqttClient struct {
	deviceId  string
	productId string
	spec      *MQTTClientSpec
	session   *clientSession
}

func (c *MqttClient) Type() codec.NetClientType {
	return codec.MQTT_CLIENT
}

func (c *MqttClient) Connect(deviceId string, network codec.NetworkConf) error {
	spec := MQTTClientSpec{}
	spec.FromJson(network.Configuration)
	if len(spec.Host) == 0 {
		return errors.New("host not be empty")
	}
	if spec.Port == 0 {
		return errors.New("port is invalidate")
	}
	if len(spec.ClientId) == 0 {
		return errors.New("clientId not be empty")
	}

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

func (c *MqttClient) Close() error {
	c.session.Disconnect()
	return nil
}
