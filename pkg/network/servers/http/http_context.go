package httpserver

import (
	"go-iot/pkg/core"
	"net/http"
)

type httpContext struct {
	core.BaseContext
	Data []byte
	r    *http.Request
}

func (ctx *httpContext) GetMessage() interface{} {
	return ctx.Data
}

func (ctx *httpContext) MsgToString() string {
	return string(ctx.Data)
}

func (ctx *httpContext) GetHeader(key string) string {
	if ctx.r == nil {
		return ""
	}
	return ctx.r.Header.Get(key)
}

func (ctx *httpContext) GetUrl() string {
	return ctx.r.RequestURI
}

func (ctx *httpContext) GetQuery(key string) string {
	return ctx.r.Form.Get(key)
}

func (ctx *httpContext) GetForm(key string) string {
	return ctx.r.Form.Get(key)
}
