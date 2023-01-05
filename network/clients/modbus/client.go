package modbus

import (
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
	devoper := codec.GetDevice(deviceId)
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
