package codec

import (
	"errors"
	"runtime/debug"

	"github.com/beego/beego/v2/core/logs"
	"github.com/dop251/goja"
)

func init() {
	RegCodecCreator(Script_Codec, func(network NetworkConf) (Codec, error) {
		codec, err := NewScriptCodec(network)
		return codec, err
	})
}

const (
	OnConnect      = "OnConnect"
	OnMessage      = "OnMessage"
	OnInvoke       = "OnInvoke"
	OnDeviceCreate = "OnDeviceCreate"
	OnDeviceDelete = "OnDeviceDelete"
	OnDeviceUpdate = "OnDeviceUpdate"
	OnStateChecker = "OnStateChecker"
	Script_Codec   = "script_codec"
)

type vmPool struct {
	chVM chan *goja.Runtime
}

func newPool(src string, size int) (*vmPool, error) {
	program, _ := goja.Compile("", src, false)
	p := vmPool{chVM: make(chan *goja.Runtime, size)}
	for i := 0; i < size; i++ {
		vm := goja.New()
		_, err := vm.RunProgram(program)
		if err != nil {
			return nil, err
		}
		console := vm.NewObject()
		console.Set("log", func(f interface{}, v ...interface{}) {
			logs.Debug(f, v...)
		})
		vm.Set("console", console)
		p.put(vm)
	}
	return &p, nil
}

func (p *vmPool) get() *goja.Runtime {
	vm := <-p.chVM
	return vm
}

func (p *vmPool) put(vm *goja.Runtime) {
	p.chVM <- vm
}

// js脚本编解码
type ScriptCodec struct {
	script    string
	productId string
	pool      *vmPool
}

func NewScriptCodec(network NetworkConf) (Codec, error) {
	pool, err := newPool(network.Script, 20)
	if err != nil {
		return nil, err
	}
	sc := &ScriptCodec{
		script:    network.Script,
		productId: network.ProductId,
		pool:      pool,
	}
	// consoleRewirte(vm)

	RegCodec(network.ProductId, sc)
	RegDeviceLifeCycle(network.ProductId, sc)

	return sc, nil
}

// 设备连接时
func (c *ScriptCodec) OnConnect(ctx MessageContext) error {
	resp := c.funcInvoke(OnConnect, ctx)
	if resp != nil {
		return nil
	}
	return errors.New("notimpl")
}

// 接收消息
func (c *ScriptCodec) OnMessage(ctx MessageContext) error {
	c.funcInvoke(OnMessage, ctx)
	return nil
}

// 命令调用
func (c *ScriptCodec) OnInvoke(ctx MessageContext) error {
	c.funcInvoke(OnInvoke, ctx)
	return nil
}

// 连接关闭
func (c *ScriptCodec) OnClose(ctx MessageContext) error {
	return nil
}

// 设备新增
func (c *ScriptCodec) OnCreate(ctx DeviceLifecycleContext) error {
	c.funcInvoke(OnDeviceCreate, ctx)
	return nil
}

// 设备删除
func (c *ScriptCodec) OnDelete(ctx DeviceLifecycleContext) error {
	c.funcInvoke(OnDeviceDelete, ctx)
	return nil
}

// 设备修改
func (c *ScriptCodec) OnUpdate(ctx DeviceLifecycleContext) error {
	c.funcInvoke(OnDeviceUpdate, ctx)
	return nil
}

// 状态检查
func (c *ScriptCodec) OnStateChecker(ctx DeviceLifecycleContext) (string, error) {
	resp := c.funcInvoke(OnStateChecker, ctx)
	if resp != nil {
		return resp.String(), nil
	}
	return "", nil
}

func (c *ScriptCodec) funcInvoke(name string, param interface{}) goja.Value {
	vm := c.pool.get()
	defer c.pool.put(vm)
	fn, success := goja.AssertFunction(vm.Get(name))
	if success {
		defer func() {
			if err := recover(); err != nil {
				logs.Error("productId: [%s], error: %v", c.productId, err)
				logs.Error(string(debug.Stack()))
			}
		}()
		resp, err := fn(goja.Undefined(), vm.ToValue(param))
		if err != nil {
			logs.Error("productId: [%s], error: %v", c.productId, err)
		}
		return resp
	}
	return nil
}
