package codec

import (
	"errors"
	"log"

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
	val, _ = vm.Get("OnMessage")
	sc.onMessage = val
	val, _ = vm.Get("OnInvoke")
	sc.onInvoke = val
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
	onMessage      otto.Value
	onInvoke       otto.Value
	onDeviceCreate otto.Value
	onDeviceDelete otto.Value
	onDeviceUpdate otto.Value
	onStateChecker otto.Value
}

// 设备连接时
func (codec *ScriptCodec) OnConnect(ctx Context) error {
	if codec.onConnect.IsDefined() {
		funcInvoke(codec.onConnect, ctx)
		return nil
	}
	return errors.New("notimpl")
}

// 接收消息
func (codec *ScriptCodec) OnMessage(ctx Context) error {
	funcInvoke(codec.onMessage, ctx)
	return nil
}

// 命令调用
func (codec *ScriptCodec) OnInvoke(ctx Context) error {
	funcInvoke(codec.onInvoke, ctx)
	return nil
}

// 连接关闭
func (codec *ScriptCodec) OnClose(ctx Context) error {
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

// 状态检查
func (codec *ScriptCodec) OnStateChecker(ctx Context) error {
	funcInvoke(codec.onStateChecker, ctx)
	return nil
}

func funcInvoke(fn otto.Value, param interface{}) {
	if fn.IsDefined() {
		_, err := fn.Call(fn, param)
		if err != nil {
			log.Fatalln(err)
		}
	}
}
