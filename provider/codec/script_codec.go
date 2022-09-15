package codec

import (
	"github.com/robertkrimen/otto"
)

func init() {
	regCodecCreator("script_codec", func(network Network) Codec {
		codec, _ := newScriptCodec(network)
		return codec
	})
}

func newScriptCodec(network Network) (Codec, error) {
	vm := otto.New()
	_, err := vm.Run(network.Script)
	sc := &ScriptCodec{
		script: network.Script,
		vm:     vm,
	}
	codecMap[network.ProductId] = sc

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

	return sc, err
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
func (codec *ScriptCodec) OnConnect(ctx Context) error {
	val, _ := codec.vm.Get("OnConnect")
	val.Call(val, ctx)
	return nil
}

// 设备解码
func (codec *ScriptCodec) Decode(ctx Context) error {
	val, _ := codec.vm.Get("Decode")
	val.Call(val, ctx)
	return nil
}

// 编码
func (codec *ScriptCodec) Encode(ctx Context) error {
	val, _ := codec.vm.Get("Encode")
	val.Call(val, ctx)
	return nil
}

// 设备新增
func (codec *ScriptCodec) OnCreate(ctx Context) error {
	val, _ := codec.vm.Get("OnDeviceCreate")
	val.Call(val, ctx)
	return nil
}

// 设备删除
func (codec *ScriptCodec) OnDelete(ctx Context) error {
	val, _ := codec.vm.Get("OnDeviceDelete")
	val.Call(val, ctx)
	return nil
}

// 设备修改
func (codec *ScriptCodec) OnUpdate(ctx Context) error {
	val, _ := codec.vm.Get("OnDeviceUpdate")
	val.Call(val, ctx)
	return nil
}

func (codec *ScriptCodec) OnStateChecker(ctx Context) error {
	val, _ := codec.vm.Get("OnStateChecker")
	val.Call(val, ctx)
	return nil
}
