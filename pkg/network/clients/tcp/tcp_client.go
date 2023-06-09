package tcpclient

import (
	"errors"
	"fmt"
	"go-iot/pkg/core"
	"go-iot/pkg/network"
	"go-iot/pkg/network/clients"
	"net"
)

func init() {
	clients.RegClient(func() network.NetClient {
		return &TcpClient{}
	})
}

type TcpClient struct {
	conn      net.Conn
	deviceId  string
	productId string
	spec      *TcpClientSpec
	session   core.Session
}

func (c *TcpClient) Type() network.NetType {
	return network.TCP_CLIENT
}

func (c *TcpClient) Connect(deviceId string, network network.NetworkConf) error {
	spec := &TcpClientSpec{}
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

	c.productId = network.ProductId

	go c.readLoop()

	return nil
}

func (c *TcpClient) readLoop() {
	session := newTcpSession(c.deviceId, c.spec, c.productId, c.conn)
	defer session.Disconnect()
	c.session = session

	sc := core.GetCodec(c.productId)

	context := &tcpContext{
		BaseContext: core.BaseContext{
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
