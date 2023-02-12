package tcpclient

import (
	"encoding/hex"
	"go-iot/pkg/core"
)

type tcpContext struct {
	core.BaseContext
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
