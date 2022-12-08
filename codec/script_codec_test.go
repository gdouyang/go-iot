package codec_test

import (
	"fmt"
	"go-iot/codec"
	"testing"

	"github.com/beego/beego/v2/core/logs"
	"github.com/robertkrimen/otto"
)

func TestOtto(t *testing.T) {
	vm := otto.New()
	vm.Run(`function test(va) {return 1}`)
	val, _ := vm.Call(`test`, nil)
	str, _ := val.ToString()
	fmt.Println("value = " + str)
	v0, _ := vm.Get("test1")
	fmt.Printf("test1 is defined = %v \n", v0.IsDefined())
	v1, _ := vm.Get("test")
	fmt.Printf("test is defined = %v \n", v1.IsDefined())
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
	c.OnConnect(&codec.MockContext{DeviceId: "fff"})
	c.OnInvoke(&codec.MockContext{DeviceId: "fff"})
	c.OnMessage(&codec.MockContext{DeviceId: "fff"})
	switch m := c.(type) {
	case codec.DeviceLifecycle:
		m.OnCreate(&codec.MockContext{DeviceId: "2222"})
	default:
	}
}
