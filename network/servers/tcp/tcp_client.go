package tcpserver

import (
	"fmt"
	"go-iot/codec"
	"net"

	"github.com/beego/beego/v2/core/logs"
)

func ClientStart(network codec.Network, call func() string) {
	spec := &TcpServerSpec{}
	spec.FromJson(network.Configuration)
	spec.Port = network.Port
	conn, err := net.Dial("tcp", spec.Host+":"+fmt.Sprint(spec.Port))
	if err != nil {
		fmt.Print(err)
	}
	codec.NewCodec(network)
	go connClientHandler(conn, network.ProductId, spec)
}

func connClientHandler(c net.Conn, productId string, spec *TcpServerSpec) {
	//1.conn是否有效
	if c == nil {
		logs.Error("无效的 socket 连接")
		return
	}
	session := newTcpSession(c)
	defer session.Disconnect()

	sc := codec.GetCodec(productId)

	context := &tcpContext{
		BaseContext: codec.BaseContext{ProductId: productId, Session: session},
	}

	sc.OnConnect(context)

	//2.新建网络数据流存储结构
	delimeter := newDelimeter(spec.Delimeter, c)

	//3.循环读取网络数据流
	for {
		//3.1 网络数据流读入 buffer
		data, err := delimeter.Read()
		//3.2 数据读尽、读取错误 关闭 socket 连接
		if err != nil {
			logs.Error("read error: " + err.Error())
			break
		}
		context.Data = data
		sc.OnMessage(context)
	}
}
