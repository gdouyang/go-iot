package core_test

import (
	"go-iot/pkg/core"
	"testing"

	logs "go-iot/pkg/logger"

	"github.com/dop251/goja"
	"github.com/stretchr/testify/assert"
)

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
function OnDeviceCreate(context) {
	console.log(JSON.stringify(context))
}
function OnDeviceDelete(context) {
	console.log(JSON.stringify(context))
}
function OnDeviceUpdate(context) {
	console.log(JSON.stringify(context))
}
function OnStateChecker(context) {
	console.log(JSON.stringify(context))
}
`
	c, err := core.NewCodec("script_codec", "test", script)
	if err != nil {
		logs.Errorf(err.Error())
	}
	c.OnConnect(&core.BaseContext{DeviceId: "fff"})
	c.OnInvoke(core.FuncInvokeContext{BaseContext: core.BaseContext{DeviceId: "fff"}})
	c.OnMessage(&core.BaseContext{DeviceId: "fff"})
	switch m := c.(type) {
	case core.DeviceLifecycle:
		m.OnCreate(&core.BaseContext{DeviceId: "2222"})
	default:
	}
}
