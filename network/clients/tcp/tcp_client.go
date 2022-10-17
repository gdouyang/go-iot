package tcpclient

import (
	"fmt"
	"go-iot/codec"
	"go-iot/network/clients"
	"net"
)

func init() {
	clients.RegClient(func() codec.NetworkClient {
		return &TcpClient{}
	})
}

type TcpClient struct {
	conn      net.Conn
	deviceId  string
	productId string
	spec      *TcpClientSpec
}

func (c *TcpClient) Type() codec.NetClientType {
	return codec.TCP_CLIENT
}

func (c *TcpClient) Connect(deviceId string, network codec.NetworkConf) error {
	spec := &TcpClientSpec{}
	spec.FromJson(network.Configuration)
	spec.Port = network.Port
	conn, err := net.Dial("tcp", spec.Host+":"+fmt.Sprint(spec.Port))
	if err != nil {
		fmt.Print(err)
		return err
	}
	c.conn = conn
	c.deviceId = deviceId
	c.spec = spec

	codec.NewCodec(network)
	c.productId = network.ProductId

	go c.readLoop()

	return nil
}

func (c *TcpClient) readLoop() {
	session := newTcpSession(c.deviceId, c.spec, c.productId, c.conn)
	defer session.Disconnect()

	sc := codec.GetCodec(c.productId)

	context := &tcpContext{
		BaseContext: codec.BaseContext{
			DeviceId:  c.deviceId,
			ProductId: c.productId,
			Session:   session},
	}

	sc.OnConnect(context)

	session.readLoop()
}

func (c *TcpClient) Reload() error {
	return nil
}

func (c *TcpClient) Stop() error {
	return nil
}
