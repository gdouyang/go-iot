package servers

type ServerMeter interface {
	// 总连接数
	TotalConnection() int32
}
