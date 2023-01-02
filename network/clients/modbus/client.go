package modbus

import (
	"fmt"
	"go-iot/codec"
	"go-iot/network/clients"
)

func init() {
	clients.RegClient(func() codec.NetClient {
		return NewClient()
	})
}

type Client struct {
}

func NewClient() *Client {
	return &Client{}
}

func (c *Client) Type() codec.NetClientType {
	return codec.MODBUS
}
func (c *Client) Connect(deviceId string, network codec.NetworkConf) error {
	spec := &ModbusSpec{}
	err := spec.FromJson(network.Configuration)
	if err != nil {
		return err
	}
	if len(spec.Protocol) == 0 {
		spec.Protocol = ProtocolTCP
	}
	if spec.Protocol != ProtocolTCP && spec.Protocol != ProtocolRTU {
		return fmt.Errorf("modbus protocol must be %s or %s", ProtocolTCP, ProtocolRTU)
	}
	session := newSession()
	session.deviceId = deviceId
	session.productId = network.ProductId
	session.conf = network.Configuration
	err = session.connection(func() {})
	if err != nil {
		return err
	}
	codec.PutSession(deviceId, session)
	session.readLoop()
	return nil
}

func (c *Client) Reload() error {
	return nil
}
func (c *Client) Close() error {
	return nil
}
