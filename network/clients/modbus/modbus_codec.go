package modbus

import (
	"go-iot/codec"
	"go-iot/network/clients"
)

func init() {
	codec.RegCodecCreator(clients.MODBUS_CODEC, func(network codec.NetworkConf) (codec.Codec, error) {
		codec, err := NewModbusScriptCodec(network)
		return codec, err
	})
}

type ModbusScriptCodec struct {
	codec.Codec
}

func NewModbusScriptCodec(network codec.NetworkConf) (codec.Codec, error) {
	c, err := codec.NewScriptCodec(network)
	if err != nil {
		return nil, err
	}
	return &ModbusScriptCodec{
		Codec: c,
	}, nil
}

// 接收消息
func (c *ModbusScriptCodec) OnMessage(ctx codec.MessageContext) error {
	c.Codec.OnMessage(ctx)
	return nil
}

// 命令调用
func (c *ModbusScriptCodec) OnInvoke(ctx codec.MessageContext) error {
	sess := ctx.GetSession()
	s := sess.(*modbusSession)
	s.connection(func() {
		c.Codec.OnInvoke(ctx)
	})
	return nil
}
