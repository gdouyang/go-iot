package modbus

import (
	"go-iot/pkg/core"
	"go-iot/pkg/network"
	"go-iot/pkg/network/clients"
)

func init() {
	clients.RegClient(func() network.NetClient {
		return NewClient()
	})
}

type Client struct {
}

func NewClient() *Client {
	return &Client{}
}

func (c *Client) Type() network.NetType {
	return network.MODBUS
}
func (c *Client) Connect(deviceId string, network network.NetworkConf) error {
	devoper := core.GetDevice(deviceId)
	tcpInfo, err := createTcpConnectionInfoByConfig(devoper)
	if err != nil {
		return err
	}
	session := newSession()
	session.deviceId = deviceId
	session.productId = network.ProductId
	session.tcpInfo = tcpInfo
	err = session.connection(func() {})
	if err != nil {
		return err
	}
	core.PutSession(deviceId, session)
	session.readLoop()
	return nil
}

func (c *Client) Reload() error {
	return nil
}
func (c *Client) Close() error {
	return nil
}
