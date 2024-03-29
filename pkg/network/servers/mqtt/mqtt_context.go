package mqttserver

import (
	"encoding/hex"
	"go-iot/pkg/core"
	"net"
	"strings"

	logs "go-iot/pkg/logger"

	"github.com/eclipse/paho.mqtt.golang/packets"
)

// auth context have no message and session, when auth pass then session will be set
type authContext struct {
	core.BaseContext
	client   *Client
	connack  *packets.ConnackPacket
	conn     net.Conn
	authFail bool
}

func (ctx *authContext) GetMessage() interface{} {
	return nil
}

func (ctx *authContext) GetSession() core.Session {
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
		device := core.GetDevice(deviceId)
		if device == nil {
			ctx._authFail(packets.ErrRefusedIDRejected)
			return
		}
		ctx.DeviceId = deviceId
		ctx.client.info.deviceId = deviceId
		ctx.authFail = false
		// after auth success, when set session will call DeviceOnline
	}
}

func (ctx *authContext) AuthFail() {
	ctx._authFail(packets.ErrRefusedNotAuthorised)
}

func (ctx *authContext) _authFail(code int) {
	ctx.authFail = true
	ctx.connack.ReturnCode = byte(code)
	err := ctx.connack.Write(ctx.conn)
	if err != nil {
		logs.Errorf("send connack to client %s failed: %v", ctx.GetClientId(), err)
	}
}

func (ctx *authContext) checkAuth() bool {
	username := ctx.GetConfig("username")
	password := ctx.GetConfig("password")
	username1 := ctx.GetUserName()
	password1 := ctx.GetPassword()
	if len(username) > 0 && username != username1 && len(password) > 0 && password != password1 {
		ctx.AuthFail()
		return false
	}
	return true
}

// mqttContext
type mqttContext struct {
	core.BaseContext
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
