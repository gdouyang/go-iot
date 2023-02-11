package mqttserver

import (
	"encoding/hex"
	"go-iot/pkg/codec"
	"net"
	"strings"

	"github.com/beego/beego/v2/core/logs"
	"github.com/eclipse/paho.mqtt.golang/packets"
)

// authContext
// auth context have no message and session
// when auth pass then session will be set
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

func (ctx *authContext) GetPassword() string {
	return ctx.client.info.password
}

func (ctx *authContext) DeviceOnline(deviceId string) {
	deviceId = strings.TrimSpace(deviceId)
	if len(deviceId) > 0 {
		device := codec.GetDevice(deviceId)
		if device == nil {
			ctx.authFail1(packets.ErrRefusedIDRejected)
			return
		}
		ctx.DeviceId = deviceId
		ctx.client.info.deviceId = deviceId
		ctx.authFail = false
		// after auth success, when set session will call DeviceOnline
	}
}

func (ctx *authContext) AuthFail() {
	ctx.authFail1(packets.ErrRefusedNotAuthorised)
}

func (ctx *authContext) authFail1(code int) {
	ctx.authFail = true
	ctx.connack.ReturnCode = byte(code)
	err := ctx.connack.Write(ctx.conn)
	if err != nil {
		logs.Error("send connack to client %s failed: %s", ctx.GetClientId(), err)
	}
}

func (ctx *authContext) checkAuth() bool {
	username := ctx.GetConfig("username")
	password := ctx.GetConfig("password")
	if len(username) > 0 && username == ctx.GetUserName() && len(password) > 0 && password == ctx.GetPassword() {
		ctx.AuthFail()
		return false
	}
	return true
}

// mqttContext
type mqttContext struct {
	codec.BaseContext
	client    *Client
	Data      []byte
	topic     string
	messageID uint16
}

func (ctx *mqttContext) GetMessage() interface{} {
	return ctx.Data
}

func (ctx *mqttContext) MsgToString() string {
	return string(ctx.Data)
}

func (ctx *mqttContext) MsgToHexStr() string {
	return hex.EncodeToString(ctx.Data)
}

func (ctx *mqttContext) Topic() string {
	return ctx.topic
}

func (ctx *mqttContext) MessageID() uint16 {
	return ctx.messageID
}

func (ctx *mqttContext) GetClientId() string {
	return ctx.client.ClientID()
}

func (ctx *mqttContext) GetUserName() string {
	return ctx.client.UserName()
}
