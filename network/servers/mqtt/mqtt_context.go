package mqttserver

import "go-iot/codec"

type mqttContext struct {
	codec.BaseContext
	Data []byte
}

func (ctx *mqttContext) GetMessage() interface{} {
	return ctx.Data
}

func (ctx *mqttContext) MsgToString() string {
	return string(ctx.Data)
}
