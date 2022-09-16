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
	sc.onConnect = val
	val, _ = vm.Get("Decode")
	sc.decode = val
	val, _ = vm.Get("Encode")
	sc.encode = val
	val, _ = vm.Get("OnDeviceCreate")
	sc.onDeviceCreate = val
	val, _ = vm.Get("OnDeviceDelete")
	sc.onDeviceDelete = val
	val, _ = vm.Get("OnDeviceUpdate")
	sc.onDeviceUpdate = val
	val, _ = vm.Get("OnStateChecker")
	sc.onStateChecker = val

	return sc, err
}

// js脚本编解码
type ScriptCodec struct {
	script         string
	vm             *otto.Otto
	onConnect      otto.Value
	decode         otto.Value
	encode         otto.Value
	onDeviceCreate otto.Value
	onDeviceDelete otto.Value
	onDeviceUpdate otto.Value
	onStateChecker otto.Value
}

// 设备连接时
func (codec *ScriptCodec) OnConnect(ctx Context) error {
	funcInvoke(codec.onConnect, ctx)
	return nil
}

// 设备解码
func (codec *ScriptCodec) Decode(ctx Context) error {
	funcInvoke(codec.decode, ctx)
	return nil
}

// 编码
func (codec *ScriptCodec) Encode(ctx Context) error {
	funcInvoke(codec.encode, ctx)
	return nil
}

// 设备新增
func (codec *ScriptCodec) OnCreate(ctx Context) error {
	funcInvoke(codec.onDeviceCreate, ctx)
	return nil
}

// 设备删除
func (codec *ScriptCodec) OnDelete(ctx Context) error {
	funcInvoke(codec.onDeviceDelete, ctx)
	return nil
}

// 设备修改
func (codec *ScriptCodec) OnUpdate(ctx Context) error {
	funcInvoke(codec.onDeviceUpdate, ctx)
	return nil
}

func (codec *ScriptCodec) OnStateChecker(ctx Context) error {
	funcInvoke(codec.onStateChecker, ctx)
	return nil
}

func funcInvoke(fn otto.Value, param interface{}) {
	if fn.IsDefined() {
		fn.Call(fn, param)
	}
}
