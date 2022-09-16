package tcpserver

import (
	"fmt"
	"go-iot/provider/codec"
	"log"
	"net"
)

func connHandler(c net.Conn, productId string) {
	//1.conn是否有效
	if c == nil {
		log.Panic("无效的 socket 连接")
	}
	session := newTcpSession(c)

	sc := codec.GetCodec(productId)

	context := &tcpContext{productId: productId, session: session}

	sc.OnConnect(context)

	//2.新建网络数据流存储结构
	buf := make([]byte, 512)
	//3.循环读取网络数据流
	for {
		//3.1 网络数据流读入 buffer
		cnt, err := c.Read(buf)
		//3.2 数据读尽、读取错误 关闭 socket 连接
		if cnt == 0 || err != nil {
			c.Close()
			break
		}

		data := buf[0:cnt]
		context.Data = data
		sc.Decode(context)
	}
}

// 开启serverSocket
func ServerSocket(network codec.Network) {

	spec := &TcpServerSpec{}
	spec.FromJson(network.Configuration)
	// 1.监听端口
	server, err := net.Listen("tcp", spec.Host+":"+fmt.Sprint(spec.Port))

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

			//并发模式 接收来自客户端的连接请求，一个连接 建立一个 conn，服务器资源有可能耗尽 BIO模式
			go connHandler(conn, network.ProductId)
		}
	}()

}
