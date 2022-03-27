package wasmhost

import "github.com/eclipse/paho.mqtt.golang/packets"

type Context interface {
	ClientID() string
	UserName() string
	Done() <-chan struct{}
}

type MqttContext struct {
	Ctx    Context
	Packet *packets.PublishPacket
}

func (c *MqttContext) ClientID() string {
	return c.Ctx.ClientID()
}

func (c *MqttContext) UserName() string {
	return c.Ctx.UserName()
}
func (c *MqttContext) Done() <-chan struct{} {
	return c.Ctx.Done()
}

func (c *MqttContext) GetPacket() *packets.PublishPacket {
	return c.Packet
}
