package modbus

import (
	"go-iot/pkg/core"
)

const MODBUS_CODEC = "modbus-script-core"

func init() {
	core.RegCodecCreator(MODBUS_CODEC, func(network core.NetworkConf) (core.Codec, error) {
		core, err := NewModbusScriptCodec(network)
		return core, err
	})
}

type ModbusScriptCodec struct {
	*core.ScriptCodec
}

func NewModbusScriptCodec(network core.NetworkConf) (core.Codec, error) {
	c, err := core.NewScriptCodec(network)
	if err != nil {
		return nil, err
	}
	sc := &ModbusScriptCodec{
		ScriptCodec: c.(*core.ScriptCodec),
	}
	core.RegCodec(network.ProductId, sc)
	core.RegDeviceLifeCycle(network.ProductId, sc)
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
	s := sess.(*modbusSession)
	modbusInvokeContext := &modbusInvokeContext{
		FuncInvokeContext: ctx,
	}
	s.connection(func() {
		c.ScriptCodec.FuncInvoke(core.OnInvoke, modbusInvokeContext)
	})
	return nil
}
