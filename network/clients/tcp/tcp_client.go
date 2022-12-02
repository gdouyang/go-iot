package tcpclient

import (
	"errors"
	"fmt"
	"go-iot/codec"
	"go-iot/network/clients"
	"net"
)

func init() {
	clients.RegClient(func() codec.NetClient {
		return &TcpClient{}
	})
}

type TcpClient struct {
	conn      net.Conn
	deviceId  string
	productId string
	spec      *TcpClientSpec
	session   codec.Session
}

func (c *TcpClient) Type() codec.NetClientType {
	return codec.TCP_CLIENT
}

func (c *TcpClient) Connect(deviceId string, network codec.NetworkConf) error {
	spec := &TcpClientSpec{}
	spec.FromJson(network.Configuration)
	if spec.Port == 0 {
		return errors.New("port must gt 0 and le 65535")
	}
	if len(spec.Host) == 0 {
		return errors.New("host must present")
	}
	conn, err := net.Dial("tcp", spec.Host+":"+fmt.Sprint(spec.Port))
	if err != nil {
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
	c.session = session

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

func (c *TcpClient) Close() error {
	c.session.Disconnect()
	return nil
}
