package mqtt5

import (
	"encoding/hex"
	"go-iot/pkg/core"
	"strings"

	"github.com/mochi-mqtt/server/v2/packets"
)

// 认证上下文没有消息和会话，当认证通过后会话将被设置
type authContext struct {
	core.BaseContext
	client       *ClientAndSession
	authFailCode byte
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
			ctx.authFailCode = packets.ErrClientIdentifierNotValid.Code
			return
		}
		ctx.DeviceId = deviceId
		ctx.client.info.deviceId = deviceId
		ctx.authFailCode = 0
		// 认证成功、让设备上线
		baseContext := &core.BaseContext{
			ProductId: ctx.client.broker.productId,
			Session:   ctx.client,
		}
		baseContext.DeviceOnline(deviceId)
	}
}

func (ctx *authContext) AuthFail() {
	ctx.authFailCode = packets.ErrBadUsernameOrPassword.Code
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
	client    *ClientAndSession
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
