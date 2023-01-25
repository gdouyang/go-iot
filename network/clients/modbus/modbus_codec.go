package modbus

import (
	"go-iot/codec"
)

const MODBUS_CODEC = "modbus-script-codec"

func init() {
	codec.RegCodecCreator(MODBUS_CODEC, func(network codec.NetworkConf) (codec.Codec, error) {
		codec, err := NewModbusScriptCodec(network)
		return codec, err
	})
}

type ModbusScriptCodec struct {
	*codec.ScriptCodec
}

func NewModbusScriptCodec(network codec.NetworkConf) (codec.Codec, error) {
	c, err := codec.NewScriptCodec(network)
	if err != nil {
		return nil, err
	}
	sc := &ModbusScriptCodec{
		ScriptCodec: c.(*codec.ScriptCodec),
	}
	codec.RegCodec(network.ProductId, sc)
	codec.RegDeviceLifeCycle(network.ProductId, sc)
	return sc, nil
}

// func (c *ModbusScriptCodec) OnConnect(ctx codec.MessageContext) error {
// 	c.ScriptCodec.OnConnect(ctx)
// 	return nil
// }

// 接收消息
// func (c *ModbusScriptCodec) OnMessage(ctx codec.MessageContext) error {
// 	c.ScriptCodec.OnMessage(ctx)
// 	return nil
// }

// 命令调用
func (c *ModbusScriptCodec) OnInvoke(ctx codec.FuncInvokeContext) error {
	sess := ctx.GetSession()
	s := sess.(*modbusSession)
	modbusInvokeContext := &modbusInvokeContext{
		FuncInvokeContext: ctx,
	}
	s.connection(func() {
		c.ScriptCodec.FuncInvoke(codec.OnInvoke, modbusInvokeContext)
	})
	return nil
}
