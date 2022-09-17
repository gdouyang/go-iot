package tcpserver

import (
	"fmt"
	"go-iot/provider/codec"
	"log"
	"net"
)

func connHandler(c net.Conn, productId string, spec *TcpServerSpec) {
	//1.conn是否有效
	if c == nil {
		log.Panic("无效的 socket 连接")
	}
	session := newTcpSession(c)

	sc := codec.GetCodec(productId)

	context := &tcpContext{productId: productId, session: session}

	sc.OnConnect(context)

	//2.新建网络数据流存储结构
	delimeter := newDelimeter(spec.Delimeter, c)
	// reader := bufio.NewReader(c)
	// var delim byte
	// var fixLengthBuf []byte
	// var scanner *bufio.Scanner
	// if spec.Delimeter.Type == DelimType_Delimited {
	// 	b := []byte(spec.Delimeter.Delimited)
	// 	delim = b[len(b)-1]
	// } else if spec.Delimeter.Type == DelimType_FixLength {
	// 	fixLengthBuf = make([]byte, spec.Delimeter.Length)
	// } else if spec.Delimeter.Type == DelimType_SplitFunc {
	// 	scanner = bufio.NewScanner(c)
	// }

	defer session.DisConnect()
	//3.循环读取网络数据流
	for {
		//3.1 网络数据流读入 buffer
		data, err := delimeter.Read()
		// var data []byte
		// var err error
		// if spec.Delimeter.Type == DelimType_Delimited {
		// 	data, err = reader.ReadSlice(delim)
		// } else if spec.Delimeter.Type == DelimType_FixLength {
		// 	var count int
		// 	count, err = reader.Read(fixLengthBuf)
		// 	data = fixLengthBuf[0:count]
		// } else if spec.Delimeter.Type == DelimType_SplitFunc {
		// 	// Create a custom split function by wrapping the existing ScanWords function.
		// 	split := func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		// 		advance, token, err = bufio.ScanWords(data, atEOF)
		// 		if err == nil && token != nil {
		// 			_, err = strconv.ParseInt(string(token), 10, 32)
		// 		}
		// 		return
		// 	}
		// 	scanner.Split(split)
		// 	for scanner.Scan() {
		// 		fmt.Printf("%s\n", scanner.Bytes())
		// 	}
		// }
		//3.2 数据读尽、读取错误 关闭 socket 连接
		if err != nil {
			log.Println("read error: " + err.Error())
			break
		}
		context.Data = data
		sc.Decode(context)
	}
}

// 开启serverSocket
func ServerSocket(network codec.Network) {

	spec := &TcpServerSpec{}
	spec.FromJson(network.Configuration)
	spec.Port = network.Port
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
			go connHandler(conn, network.ProductId, spec)
		}
	}()

}
