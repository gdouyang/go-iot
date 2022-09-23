package tcpserver

import (
	"fmt"
	"go-iot/provider/codec"
	"net"

	"github.com/beego/beego/v2/core/logs"
)

// 开启serverSocket
func ServerSocket(network codec.Network) {

	spec := &TcpServerSpec{}
	spec.FromJson(network.Configuration)
	spec.Port = network.Port
	// 1.监听端口
	addr := spec.Host + ":" + fmt.Sprint(spec.Port)
	server, err := net.Listen("tcp", addr)

	codec.NewCodec(network)

	if err != nil {
		fmt.Println("开启socket服务失败")
	}
	go func() {
		for {
			//2.接收来自 client 的连接,会阻塞
			conn, err := server.Accept()

			if err != nil {
				fmt.Println("连接出错")
			}

			//并发模式 接收来自客户端的连接请求，一个连接 建立一个 conn
			go connHandler(conn, network.ProductId, spec)
		}
	}()
}

func connHandler(c net.Conn, productId string, spec *TcpServerSpec) {
	//1.conn是否有效
	if c == nil {
		logs.Error("无效的 socket 连接")
		return
	}
	session := newTcpSession(c)
	defer session.DisConnect()

	sc := codec.GetCodec(productId)

	context := &tcpContext{productId: productId, session: session}

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
		sc.Decode(context)
	}
}
