package websocketsocker

import (
	"encoding/hex"
	"go-iot/codec"
	"net/http"
	"net/url"

	"github.com/gorilla/websocket"
)

type websocketContext struct {
	codec.BaseContext
	Data       []byte
	msgType    int
	header     http.Header
	form       url.Values
	requestURI string
}

func (ctx *websocketContext) GetMessage() interface{} {
	return ctx.Data
}

func (ctx *websocketContext) MsgToString() string {
	return string(ctx.Data)
}

func (ctx *websocketContext) MsgToHexStr() string {
	return hex.EncodeToString(ctx.Data)
}

func (ctx *websocketContext) IsTextMessage() bool {
	return ctx.msgType == websocket.TextMessage
}

func (ctx *websocketContext) IsBinaryMessage() bool {
	return ctx.msgType == websocket.BinaryMessage
}

func (ctx *websocketContext) GetHeader(key string) string {
	if ctx.header == nil {
		return ""
	}
	return ctx.header.Get(key)
}

func (ctx *websocketContext) GetUrl() string {
	return ctx.requestURI
}

func (ctx *websocketContext) GetQuery(key string) string {
	return ctx.form.Get(key)
}

func (ctx *websocketContext) GetForm(key string) string {
	return ctx.form.Get(key)
}
