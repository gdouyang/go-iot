package mqttserver

import (
	"go-iot/codec"
	"net"
	"strings"

	"github.com/beego/beego/v2/core/logs"
	MQTT "github.com/eclipse/paho.mqtt.golang"
	"github.com/eclipse/paho.mqtt.golang/packets"
)

// authContext
type authContext struct {
	codec.BaseContext
	client   *Client
	connack  *packets.ConnackPacket
	conn     net.Conn
	authFail bool
}

func (ctx *authContext) GetMessage() interface{} {
	return nil
}

func (ctx *authContext) GetSession() codec.Session {
	return nil
}

func (ctx *authContext) GetClientId() string {
	return ctx.client.ClientID()
}

func (ctx *authContext) GetUserName() string {
	return ctx.client.UserName()
}

func (ctx *authContext) DeviceOnline(deviceId string) {
	deviceId = strings.TrimSpace(deviceId)
	if len(deviceId) > 0 {
		ctx.DeviceId = deviceId
		ctx.client.info.deviceId = deviceId
		ctx.authFail = false
	}
}

func (ctx *authContext) AuthFail() {
	ctx.authFail = true
	ctx.connack.ReturnCode = packets.ErrRefusedNotAuthorised
	err := ctx.connack.Write(ctx.conn)
	if err != nil {
		logs.Error("send connack to client %s failed: %s", ctx.GetClientId(), err)
	}
}

func (ctx *authContext) checkAuth() bool {
	username := ctx.GetConfig("username")
	password := ctx.GetConfig("password")
	if username != nil && username == ctx.GetUserName() && password != nil && password == ctx.client.info.password {
		ctx.AuthFail()
		return false
	}
	return true
}

// mqttContext
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
