package core

import (
	"errors"
	"fmt"
	"runtime/debug"

	"go-iot/pkg/core/util"
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
	OnConnect          = "OnConnect"
	OnMessage          = "OnMessage"
	OnInvoke           = "OnInvoke"
	On_Device_Deploy   = "OnDeviceDeploy"
	On_Device_UnDeploy = "OnDeviceUnDeploy"
	On_State_Checker   = "OnStateChecker"
	Script_Codec       = "script_codec"
)

type globe struct {
}

// crc16
func (g globe) ToCrc16Str(str string) string {
	d := util.ToCrc16Str(str)
	return d
}

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
		vm.Set("globe", globe{})
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
	_, err := c.FuncInvoke(OnConnect, ctx)
	return err
}

// 接收消息
func (c *ScriptCodec) OnMessage(ctx MessageContext) error {
	_, err := c.FuncInvoke(OnMessage, ctx)
	return err
}

// 命令调用
func (c *ScriptCodec) OnInvoke(ctx FuncInvokeContext) error {
	_, err := c.FuncInvoke(OnInvoke, ctx)
	return err
}

// 连接关闭
func (c *ScriptCodec) OnClose(ctx MessageContext) error {
	return nil
}

// 设备新增
func (c *ScriptCodec) OnDeviceDeploy(ctx DeviceLifecycleContext) error {
	_, err := c.FuncInvoke(On_Device_Deploy, ctx)
	return err
}

// 设备修改
func (c *ScriptCodec) OnDeviceUnDeploy(ctx DeviceLifecycleContext) error {
	_, err := c.FuncInvoke(On_Device_UnDeploy, ctx)
	return err
}

// 状态检查
func (c *ScriptCodec) OnStateChecker(ctx DeviceLifecycleContext) (string, error) {
	resp, err := c.FuncInvoke(On_State_Checker, ctx)
	if resp != nil {
		return resp.String(), nil
	}
	return "", err
}

func (c *ScriptCodec) FuncInvoke(name string, param interface{}) (resp goja.Value, err error) {
	vm := c.pool.Get()
	defer c.pool.Put(vm)
	fn, success := goja.AssertFunction(vm.Get(name))
	if success {
		defer func() {
			if rec := recover(); rec != nil {
				logs.Errorf("productId: [%s], error: %v", c.productId, rec)
				logs.Errorf(string(debug.Stack()))
				err = fmt.Errorf("%v", rec)
				resp = goja.Undefined()
			}
		}()
		resp, err = fn(goja.Undefined(), vm.ToValue(param))
		if err != nil {
			logs.Errorf("productId: [%s], error: %v", c.productId, err)
		}
		return resp, err
	}
	return nil, ErrNotImpl
}
