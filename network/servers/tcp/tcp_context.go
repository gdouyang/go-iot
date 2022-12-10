package tcpserver

import (
	"encoding/hex"
	"go-iot/codec"
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

func (ctx *tcpContext) HexMsg() string {
	return hex.EncodeToString(ctx.Data)
}
