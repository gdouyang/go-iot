package websocketsocker

import (
	"go-iot/codec"
	"net/http"

	"github.com/gorilla/websocket"
)

type websocketContext struct {
	codec.BaseContext
	Data    []byte
	msgType int
	r       *http.Request
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

func (ctx *websocketContext) GetHeader(key string) string {
	if ctx.r == nil {
		return ""
	}
	return ctx.r.Header.Get(key)
}

func (ctx *websocketContext) GetUrl() string {
	return ctx.r.RequestURI
}

func (ctx *websocketContext) GetQuery(key string) string {
	return ctx.r.Form.Get(key)
}

func (ctx *websocketContext) GetForm(key string) string {
	return ctx.r.Form.Get(key)
}
