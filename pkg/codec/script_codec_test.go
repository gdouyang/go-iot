package codec_test

import (
	"go-iot/pkg/codec"
	"testing"

	"github.com/beego/beego/v2/core/logs"
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
	var network codec.NetworkConf = codec.NetworkConf{
		ProductId: "test",
		CodecId:   "script_codec",
		Script: `
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
`,
	}
	c, err := codec.NewCodec(network)
	if err != nil {
		logs.Error(err)
	}
	c.OnConnect(&codec.BaseContext{DeviceId: "fff"})
	c.OnInvoke(codec.FuncInvokeContext{BaseContext: codec.BaseContext{DeviceId: "fff"}})
	c.OnMessage(&codec.BaseContext{DeviceId: "fff"})
	switch m := c.(type) {
	case codec.DeviceLifecycle:
		m.OnCreate(&codec.BaseContext{DeviceId: "2222"})
	default:
	}
}
