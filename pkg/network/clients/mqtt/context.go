package mqttclient

import (
	"encoding/hex"
	"go-iot/pkg/core"

	MQTT "github.com/eclipse/paho.mqtt.golang"
)

type mqttClientContext struct {
	core.BaseContext
	Data MQTT.Message
}

func (ctx *mqttClientContext) GetMessage() interface{} {
	return ctx.Data.Payload()
}

func (ctx *mqttClientContext) MsgToString() string {
	return string(ctx.Data.Payload())
}

func (ctx *mqttClientContext) MsgToHexStr() string {
	return hex.EncodeToString(ctx.Data.Payload())
}

func (ctx *mqttClientContext) Topic() string {
	return ctx.Data.Topic()
}

func (ctx *mqttClientContext) MessageID() uint16 {
	return ctx.Data.MessageID()
}

func (ctx *mqttClientContext) GetClientId() string {
	return ctx.Session.(*MqttClientSession).ClientID
}

func (ctx *mqttClientContext) GetUserName() string {
	return ctx.Session.(*MqttClientSession).Username
}
