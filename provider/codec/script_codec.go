package codec

import (
	"github.com/robertkrimen/otto"
)

func NewScriptCodec(productId, script string) error {
	vm := otto.New()
	_, err := vm.Run(script)
	sc := &ScriptCodec{
		script: script,
		vm:     vm,
	}
	codecMap[productId] = sc

	var val, _ = vm.Get("OnConnect")
	sc.hasOnConnect = val.IsDefined()
	val, _ = vm.Get("Decode")
	sc.hasDecode = val.IsDefined()
	val, _ = vm.Get("Encode")
	sc.hasEncode = val.IsDefined()
	val, _ = vm.Get("OnDeviceCreate")
	sc.hasOnDeviceCreate = val.IsDefined()
	val, _ = vm.Get("OnDeviceDelete")
	sc.hasOnDeviceDelete = val.IsDefined()
	val, _ = vm.Get("OnDeviceUpdate")
	sc.hasOnDeviceUpdate = val.IsDefined()
	val, _ = vm.Get("OnStateChecker")
	sc.hasOnStateChecker = val.IsDefined()

	return err
}

// js脚本编解码
type ScriptCodec struct {
	script            string
	vm                *otto.Otto
	hasOnConnect      bool
	hasDecode         bool
	hasEncode         bool
	hasOnDeviceCreate bool
	hasOnDeviceDelete bool
	hasOnDeviceUpdate bool
	hasOnStateChecker bool
}

// 设备连接时
func (codec *ScriptCodec) OnConnect(ctx *Context) error {
	codec.vm.Call("OnConnect", ctx)
	return nil
}

// 设备解码
func (codec *ScriptCodec) Decode(ctx *Context) error {
	codec.vm.Call("Decode", ctx)
	return nil
}

// 编码
func (codec *ScriptCodec) Encode(ctx *Context) error {
	codec.vm.Call("Encode", ctx)
	return nil
}

// 设备新增
func (codec *ScriptCodec) OnDeviceCreate(ctx *Context) error {
	codec.vm.Call("OnDeviceCreate", ctx)
	return nil
}

// 设备删除
func (codec *ScriptCodec) OnDeviceDelete(ctx *Context) error {
	codec.vm.Call("OnDeviceDelete", ctx)
	return nil
}

// 设备修改
func (codec *ScriptCodec) OnDeviceUpdate(ctx *Context) error {
	codec.vm.Call("OnDeviceUpdate", ctx)
	return nil
}

func (codec *ScriptCodec) OnStateChecker(ctx *Context) error {
	codec.vm.Call("OnStateChecker", ctx)
	return nil
}
