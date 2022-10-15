package tcpclient

import (
	"fmt"
	"go-iot/codec"
	"net"
)

func ClientStart(deviceId string, network codec.Network) bool {
	spec := &TcpClientSpec{}
	spec.FromJson(network.Configuration)
	spec.Port = network.Port
	conn, err := net.Dial("tcp", spec.Host+":"+fmt.Sprint(spec.Port))
	if err != nil {
		fmt.Print(err)
		return false
	}
	codec.NewCodec(network)
	productId := network.ProductId
	go func() {
		session := newTcpSession(deviceId, spec, productId, conn)
		defer session.Disconnect()

		sc := codec.GetCodec(productId)

		context := &tcpContext{
			BaseContext: codec.BaseContext{
				DeviceId:  deviceId,
				ProductId: productId,
				Session:   session},
		}

		sc.OnConnect(context)

		session.readLoop()
	}()
	return true
}
