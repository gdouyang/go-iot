// 编解码
package codec

import (
	"fmt"
	"runtime/debug"

	"go-iot/pkg/core"
	logs "go-iot/pkg/logger"

	"github.com/dop251/goja"
)

func init() {
	core.RegCodecCreator(core.Script_Codec, func(productId, script string) (core.Codec, error) {
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
)

// js脚本编解码
type ScriptCodec struct {
	script    string
	productId string
	pool      *VmPool
}

func NewScriptCodec(productId, script string) (core.Codec, error) {
	// 关闭旧VmPool
	oldCodec := core.GetCodec(productId)
	if c, ok := oldCodec.(*ScriptCodec); ok {
		c.pool.Close()
	}
	// 创建新的VmPool
	pool, err := NewVmPool1(script, 20, productId)
	if err != nil {
		return nil, err
	}
	sc := &ScriptCodec{
		script:    script,
		productId: productId,
		pool:      pool,
	}

	core.RegCodec(productId, sc)
	core.RegDeviceLifeCycle(productId, sc)

	return sc, nil
}

// 设备连接时
func (c *ScriptCodec) OnConnect(ctx core.MessageContext) error {
	_, err := c.FuncInvoke(OnConnect, ctx)
	return err
}

// 接收消息
func (c *ScriptCodec) OnMessage(ctx core.MessageContext) error {
	_, err := c.FuncInvoke(OnMessage, ctx)
	return err
}

// 命令调用
func (c *ScriptCodec) OnInvoke(ctx core.FuncInvokeContext) error {
	_, err := c.FuncInvoke(OnInvoke, ctx)
	return err
}

// 连接关闭
func (c *ScriptCodec) OnClose(ctx core.MessageContext) error {
	return nil
}

// 设备新增
func (c *ScriptCodec) OnDeviceDeploy(ctx core.DeviceLifecycleContext) error {
	_, err := c.FuncInvoke(On_Device_Deploy, ctx)
	return err
}

// 设备修改
func (c *ScriptCodec) OnDeviceUnDeploy(ctx core.DeviceLifecycleContext) error {
	_, err := c.FuncInvoke(On_Device_UnDeploy, ctx)
	return err
}

// 状态检查
func (c *ScriptCodec) OnStateChecker(ctx core.DeviceLifecycleContext) (string, error) {
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
				l := fmt.Sprintf("productId: [%s] error: %v", c.productId, rec)
				logs.Errorf(l)
				deviceId := ""
				if ctx, ok := param.(core.DeviceLifecycleContext); ok && ctx.GetDevice() != nil {
					deviceId = ctx.GetDevice().Id
				}
				core.DebugLog(deviceId, c.productId, l)
				logs.Errorf(string(debug.Stack()))
				err = fmt.Errorf("%v", rec)
				resp = goja.Undefined()
			}
		}()
		resp, err = fn(goja.Undefined(), vm.ToValue(param))
		if err != nil {
			logs.Errorf("productId: [%s], error: %v", c.productId, err)
			deviceId := ""
			if ctx, ok := param.(core.DeviceLifecycleContext); ok && ctx.GetDevice() != nil {
				deviceId = ctx.GetDevice().Id
			}
			core.DebugLog(deviceId, c.productId, err.Error())
		}
		return resp, err
	}
	return nil, core.ErrFunctionNotImpl
}
