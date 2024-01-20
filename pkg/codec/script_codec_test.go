package codec_test

import (
	_ "go-iot/pkg/codec"
	"go-iot/pkg/core"
	"go-iot/pkg/logger"
	"testing"

	"github.com/dop251/goja"
	"github.com/stretchr/testify/assert"
)

func init() {
	logger.InitNop()
}

func TestOtto(t *testing.T) {
	vm := goja.New()
	vm.RunString(`function test(va) {return 1}`)
	fn, success := goja.AssertFunction(vm.Get("test"))
	assert.True(t, success)
	val, err1 := fn(goja.Undefined())
	assert.Nil(t, err1)
	assert.Equal(t, int64(1), val.ToInteger())
	_, success = goja.AssertFunction(vm.Get("test1"))
	assert.False(t, success)
	stacks := vm.CaptureCallStack(0, nil)
	for _, v := range stacks {
		logger.Errorf("%s %v", v.FuncName(), v.Position())
	}
}

func TestDecode(t *testing.T) {
	script := `
function OnConnect(context) {
  console.log(JSON.stringify(context))
}
function OnMessage(context) {
  console.log("122")
  console.log(JSON.stringify(context))
}
function OnInvoke(context) {
	console.log(JSON.stringify(context))
}
function OnDeviceDeploy(context) {
	console.log(JSON.stringify(context))
}
function OnDeviceUnDeploy(context) {
	console.log(JSON.stringify(context))
}
function OnStateChecker(context) {
	console.log(JSON.stringify(context))
}
`
	c, err := core.NewCodec("script_codec", "test", script)
	if err != nil {
		logger.Errorf(err.Error())
	}
	c.OnConnect(&core.BaseContext{DeviceId: "fff"})
	c.OnInvoke(core.FuncInvokeContext{BaseContext: core.BaseContext{DeviceId: "fff"}})
	c.OnMessage(&core.BaseContext{DeviceId: "fff"})
	switch m := c.(type) {
	case core.DeviceLifecycle:
		m.OnDeviceDeploy(&core.BaseContext{DeviceId: "2222"})
	default:
	}
}

func TestDecodeErr(t *testing.T) {
	logger.InitNop()
	script := `
function OnConnect(context) {
	context.getMessage()
  console.log(JSON.stringify(context))
}
`
	c, err := core.NewCodec("script_codec", "test", script)
	assert.Nil(t, err)
	core.RegCodec("test", c)
	err = c.OnMessage(&core.BaseContext{DeviceId: "fff"})
	assert.Equal(t, core.ErrFunctionNotImpl, err)
}
func TestHttp1(t *testing.T) {
	logger.InitNop()
	script := `
function OnConnect(context) {
	var resp = globe.HttpRequest({
		"method": "get",
		"url": "http://www.baidu.com"
	})
	console.log(resp)
}
`
	c, err := core.NewCodec("script_codec", "test", script)
	assert.Nil(t, err)
	err = c.OnConnect(&core.BaseContext{DeviceId: "fff"})
	assert.Nil(t, err)
}
