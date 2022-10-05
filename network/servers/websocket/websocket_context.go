package websocketsocker

import (
	"go-iot/codec"

	"github.com/gorilla/websocket"
)

type websocketContext struct {
	codec.BaseContext
	Data    []byte
	msgType int
}

func (ctx *websocketContext) GetMessage() interface{} {
	return ctx.Data
}

func (ctx *websocketContext) MsgToString() string {
	return string(ctx.Data)
}

func (ctx *websocketContext) IsTextMessage() bool {
	return ctx.msgType == websocket.TextMessage
}

func (ctx *websocketContext) IsBinaryMessage() bool {
	return ctx.msgType == websocket.BinaryMessage
}
