package mqttclient

import (
	"go-iot/codec"

	MQTT "github.com/eclipse/paho.mqtt.golang"
)

type mqttClientContext struct {
	codec.BaseContext
	Data MQTT.Message
}

func (ctx *mqttClientContext) GetMessage() interface{} {
	return ctx.Data
}

func (ctx *mqttClientContext) MsgToString() string {
	return string(ctx.Data.Payload())
}

func (ctx *mqttClientContext) Topic() string {
	return ctx.Data.Topic()
}

func (ctx *mqttClientContext) MessageID() uint16 {
	return ctx.Data.MessageID()
}

func (ctx *mqttClientContext) GetClientId() string {
	return ctx.Session.(*ClientSession).ClientID
}

func (ctx *mqttClientContext) GetUserName() string {
	return ctx.Session.(*ClientSession).Username
}
