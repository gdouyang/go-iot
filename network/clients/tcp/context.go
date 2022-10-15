package tcpclient

import "go-iot/codec"

type tcpContext struct {
	codec.BaseContext
	Data []byte
}

func (ctx *tcpContext) GetMessage() interface{} {
	return ctx.Data
}

func (ctx *tcpContext) MsgToString() string {
	return string(ctx.Data)
}
