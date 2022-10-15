package tcpclient

import (
	"fmt"
	"go-iot/codec"
	"net"
)

func ClientStart(network codec.Network) bool {
	spec := &TcpClientSpec{}
	spec.FromJson(network.Configuration)
	spec.Port = network.Port
	conn, err := net.Dial("tcp", spec.Host+":"+fmt.Sprint(spec.Port))
	if err != nil {
		fmt.Print(err)
		return false
	}
	codec.NewCodec(network)
	go connClientHandler(conn, network.ProductId, spec)
	return true
}

func connClientHandler(conn net.Conn, productId string, spec *TcpClientSpec) {
	session := newTcpSession(spec, productId, conn)
	defer session.Disconnect()

	sc := codec.GetCodec(productId)

	context := &tcpContext{
		BaseContext: codec.BaseContext{ProductId: productId, Session: session},
	}

	sc.OnConnect(context)

	session.readLoop()
}
