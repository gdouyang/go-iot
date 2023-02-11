package tcpclient

import (
	"encoding/hex"
	"go-iot/pkg/codec"
)

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

func (ctx *tcpContext) MsgToHexStr() string {
	return hex.EncodeToString(ctx.Data)
}
