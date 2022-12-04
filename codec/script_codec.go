package codec

import (
	"errors"
	"runtime/debug"

	"github.com/beego/beego/v2/core/logs"
	"github.com/robertkrimen/otto"
)

func init() {
	regCodecCreator(CodecIdScriptCode, func(network NetworkConf) Codec {
		codec, _ := newScriptCodec(network)
		return codec
	})
}

const (
	OnConnect         = "OnConnect"
	OnMessage         = "OnMessage"
	OnInvoke          = "OnInvoke"
	OnDeviceCreate    = "OnDeviceCreate"
	OnDeviceDelete    = "OnDeviceDelete"
	OnDeviceUpdate    = "OnDeviceUpdate"
	OnStateChecker    = "OnStateChecker"
	CodecIdScriptCode = "script_codec"
)

func newScriptCodec(network NetworkConf) (Codec, error) {
	vm := otto.New()
	_, err := vm.Run(network.Script)
	sc := &ScriptCodec{
		script: network.Script,
		vm:     vm,
	}
	RegCodec(network.ProductId, sc)
	regDeviceLifeCycle(CodecIdScriptCode, sc)

	var val, _ = vm.Get(OnConnect)
	sc.onConnect = val.IsDefined()
	val, _ = vm.Get(OnMessage)
	sc.onMessage = val.IsDefined()
	val, _ = vm.Get(OnInvoke)
	sc.onInvoke = val.IsDefined()
	val, _ = vm.Get(OnDeviceCreate)
	sc.onDeviceCreate = val.IsDefined()
	val, _ = vm.Get(OnDeviceDelete)
	sc.onDeviceDelete = val.IsDefined()
	val, _ = vm.Get(OnDeviceUpdate)
	sc.onDeviceUpdate = val.IsDefined()
	val, _ = vm.Get(OnStateChecker)
	sc.onStateChecker = val.IsDefined()

	return sc, err
}

// js脚本编解码
type ScriptCodec struct {
	script         string
	vm             *otto.Otto
	onConnect      bool
	onMessage      bool
	onInvoke       bool
	onDeviceCreate bool
	onDeviceDelete bool
	onDeviceUpdate bool
	onStateChecker bool
}

// 设备连接时
func (c *ScriptCodec) OnConnect(ctx Context) error {
	if c.onConnect {
		c.funcInvoke(OnConnect, ctx)
		return nil
	}
	return errors.New("notimpl")
}

// 接收消息
func (c *ScriptCodec) OnMessage(ctx Context) error {
	if c.onMessage {
		c.funcInvoke(OnMessage, ctx)
	}
	return nil
}

// 命令调用
func (c *ScriptCodec) OnInvoke(ctx Context) error {
	if c.onInvoke {
		c.funcInvoke(OnInvoke, ctx)
	}
	return nil
}

// 连接关闭
func (c *ScriptCodec) OnClose(ctx Context) error {
	return nil
}

// 设备新增
func (c *ScriptCodec) OnCreate(ctx DeviceLifecycleContext) error {
	if c.onDeviceCreate {
		c.funcInvoke(OnDeviceCreate, ctx)
	}
	return nil
}

// 设备删除
func (c *ScriptCodec) OnDelete(ctx DeviceLifecycleContext) error {
	if c.onDeviceDelete {
		c.funcInvoke(OnDeviceDelete, ctx)
	}
	return nil
}

// 设备修改
func (c *ScriptCodec) OnUpdate(ctx DeviceLifecycleContext) error {
	if c.onDeviceUpdate {
		c.funcInvoke(OnDeviceUpdate, ctx)
	}
	return nil
}

// 状态检查
func (c *ScriptCodec) OnStateChecker(ctx DeviceLifecycleContext) (string, error) {
	if c.onStateChecker {
		resp := c.funcInvoke(OnStateChecker, ctx)
		return resp.ToString()
	}
	return "", nil
}

func (c *ScriptCodec) funcInvoke(name string, param interface{}) otto.Value {
	vm := c.vm.Copy()
	fn, _ := vm.Get(name)
	if fn.IsDefined() {
		defer func() {
			if err := recover(); err != nil {
				logs.Error(err)
				logs.Error(string(debug.Stack()))
			}
		}()
		// logs.Warn(fmt.Sprintf("%p", &fn))
		resp, err := fn.Call(otto.Value{}, param)
		if err != nil {
			logs.Error(err)
		}
		return resp
	}
	return otto.Value{}
}
