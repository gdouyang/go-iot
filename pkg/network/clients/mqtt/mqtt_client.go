package mqttclient

import (
	"errors"
	"go-iot/pkg/core"
	"go-iot/pkg/network"
	"go-iot/pkg/network/clients"
)

func init() {
	clients.RegClient(func() network.NetClient {
		return &MqttClient{}
	})
}

type MqttClient struct {
	deviceId  string
	productId string
	spec      *MQTTClientSpec
	session   *MqttClientSession
}

func (c *MqttClient) Type() network.NetType {
	return network.MQTT_CLIENT
}

func (c *MqttClient) Connect(deviceId string, network network.NetworkConf) error {
	spec := MQTTClientSpec{}
	err := spec.FromNetwork(network)
	if err != nil {
		return err
	}
	devoper := core.GetDevice(deviceId)
	if devoper == nil {
		return errors.New("devoper is nil")
	}
	err = spec.SetByConfig(devoper)
	if err != nil {
		return err
	}
	if len(spec.Host) == 0 {
		return errors.New("host must be present")
	}
	if spec.Port == 0 {
		return errors.New("port is invalidate")
	}
	if len(spec.ClientId) == 0 {
		return errors.New("clientId must be present")
	}

	session, err := newClientSession(deviceId, network, &spec)
	if err != nil {
		return err
	}

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
