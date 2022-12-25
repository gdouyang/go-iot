package modbus

import (
	"encoding/hex"
	"go-iot/codec"
)

type context struct {
	codec.BaseContext
	Data []byte
}

func (ctx *context) GetMessage() interface{} {
	return ctx.Data
}

func (ctx *context) MsgToString() string {
	return string(ctx.Data)
}

func (ctx *context) HexMsg() string {
	return hex.EncodeToString(ctx.Data)
}
