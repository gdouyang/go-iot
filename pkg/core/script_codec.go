package core

import (
	"errors"
	"runtime/debug"

	logs "go-iot/pkg/logger"

	"github.com/dop251/goja"
)

func init() {
	RegCodecCreator(Script_Codec, func(productId, script string) (Codec, error) {
		core, err := NewScriptCodec(productId, script)
		return core, err
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

// javascript vm pool
type VmPool struct {
	chVM chan *goja.Runtime
}

// new a vm pool
func NewVmPool(src string, size int) (*VmPool, error) {
	if len(src) == 0 {
		return nil, errors.New("script must be present")
	}
	program, err := goja.Compile("", src, false)
	if err != nil {
		return nil, err
	}
	p := VmPool{chVM: make(chan *goja.Runtime, size)}
	for i := 0; i < size; i++ {
		vm := goja.New()
		_, err := vm.RunProgram(program)
		if err != nil {
			return nil, err
		}
		console := vm.NewObject()
		console.Set("log", func(v ...interface{}) {
			logs.Debugf("%v", v...)
		})
		vm.Set("console", console)
		p.Put(vm)
	}
	return &p, nil
}

func (p *VmPool) Get() *goja.Runtime {
	vm := <-p.chVM
	return vm
}

func (p *VmPool) Put(vm *goja.Runtime) {
	p.chVM <- vm
}

// js脚本编解码
type ScriptCodec struct {
	script    string
	productId string
	pool      *VmPool
}

func NewScriptCodec(productId, script string) (Codec, error) {
	pool, err := NewVmPool(script, 20)
	if err != nil {
		return nil, err
	}
	sc := &ScriptCodec{
		script:    script,
		productId: productId,
		pool:      pool,
	}

	RegCodec(productId, sc)
	RegDeviceLifeCycle(productId, sc)

	return sc, nil
}

// 设备连接时
func (c *ScriptCodec) OnConnect(ctx MessageContext) error {
	resp := c.FuncInvoke(OnConnect, ctx)
	if resp != nil {
		return nil
	}
	return ErrNotImpl
}

// 接收消息
func (c *ScriptCodec) OnMessage(ctx MessageContext) error {
	c.FuncInvoke(OnMessage, ctx)
	return nil
}

// 命令调用
func (c *ScriptCodec) OnInvoke(ctx FuncInvokeContext) error {
	c.FuncInvoke(OnInvoke, ctx)
	return nil
}

// 连接关闭
func (c *ScriptCodec) OnClose(ctx MessageContext) error {
	return nil
}

// 设备新增
func (c *ScriptCodec) OnCreate(ctx DeviceLifecycleContext) error {
	c.FuncInvoke(OnDeviceCreate, ctx)
	return nil
}

// 设备删除
func (c *ScriptCodec) OnDelete(ctx DeviceLifecycleContext) error {
	c.FuncInvoke(OnDeviceDelete, ctx)
	return nil
}

// 设备修改
func (c *ScriptCodec) OnUpdate(ctx DeviceLifecycleContext) error {
	c.FuncInvoke(OnDeviceUpdate, ctx)
	return nil
}

// 状态检查
func (c *ScriptCodec) OnStateChecker(ctx DeviceLifecycleContext) (string, error) {
	resp := c.FuncInvoke(OnStateChecker, ctx)
	if resp != nil {
		return resp.String(), nil
	}
	return "", nil
}

func (c *ScriptCodec) FuncInvoke(name string, param interface{}) goja.Value {
	vm := c.pool.Get()
	defer c.pool.Put(vm)
	fn, success := goja.AssertFunction(vm.Get(name))
	if success {
		defer func() {
			if err := recover(); err != nil {
				logs.Errorf("productId: [%s], error: %v", c.productId, err)
				logs.Errorf(string(debug.Stack()))
			}
		}()
		resp, err := fn(goja.Undefined(), vm.ToValue(param))
		if err != nil {
			logs.Errorf("productId: [%s], error: %v", c.productId, err)
		}
		return resp
	}
	return nil
}
