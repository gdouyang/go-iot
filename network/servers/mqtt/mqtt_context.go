package mqttserver

import (
	"go-iot/codec"
	"net"

	"github.com/beego/beego/v2/core/logs"
	MQTT "github.com/eclipse/paho.mqtt.golang"
	"github.com/eclipse/paho.mqtt.golang/packets"
)

type mqttContext struct {
	codec.BaseContext
	Data   []byte
	client *Client
}

func (ctx *mqttContext) GetMessage() interface{} {
	return ctx.Data
}

func (ctx *mqttContext) MsgToString() string {
	return string(ctx.Data)
}

func (ctx *mqttContext) GetClientId() string {
	return ctx.client.ClientID()
}

func (ctx *mqttContext) GetUserName() string {
	return ctx.client.UserName()
}

func (ctx *mqttContext) checkAuth(connack *packets.ConnackPacket, conn net.Conn) bool {
	username := ctx.GetConfig("username")
	password := ctx.GetConfig("password")
	if username != nil && username == ctx.GetUserName() && password != nil && password == ctx.client.info.password {
		connack.ReturnCode = packets.ErrRefusedNotAuthorised
		err := connack.Write(conn)
		if err != nil {
			logs.Error("send connack to client %s failed: %s", ctx.GetClientId(), err)
		}
		ctx.client.closeAndDelSession()
		return false
	}
	return true
}

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
