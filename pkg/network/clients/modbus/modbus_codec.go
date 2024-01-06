package modbus

import (
	"go-iot/pkg/codec"
	"go-iot/pkg/core"
)

const MODBUS_CODEC = "modbus-script-core"

func init() {
	core.RegCodecCreator(MODBUS_CODEC, func(productId string, script string) (core.Codec, error) {
		core, err := NewModbusScriptCodec(productId, script)
		return core, err
	})
}

type ModbusScriptCodec struct {
	*codec.ScriptCodec
}

func NewModbusScriptCodec(productId string, script string) (core.Codec, error) {
	c, err := codec.NewScriptCodec(productId, script)
	if err != nil {
		return nil, err
	}
	sc := &ModbusScriptCodec{
		ScriptCodec: c.(*codec.ScriptCodec),
	}
	core.RegCodec(productId, sc)
	core.RegDeviceLifeCycle(productId, sc)
	return sc, nil
}

// func (c *ModbusScriptCodec) OnConnect(ctx core.MessageContext) error {
// 	c.ScriptCodec.OnConnect(ctx)
// 	return nil
// }

// 接收消息
// func (c *ModbusScriptCodec) OnMessage(ctx core.MessageContext) error {
// 	c.ScriptCodec.OnMessage(ctx)
// 	return nil
// }

// 命令调用
func (c *ModbusScriptCodec) OnInvoke(ctx core.FuncInvokeContext) error {
	sess := ctx.GetSession()
	s := sess.(*ModbusSession)
	modbusInvokeContext := &modbusInvokeContext{
		FuncInvokeContext: ctx,
	}
	s.connection(func() {
		c.ScriptCodec.FuncInvoke(codec.OnInvoke, modbusInvokeContext)
	})
	return nil
}
