package codec_test

import (
	"fmt"
	"go-iot/codec"
	"testing"

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
	var network codec.Network = codec.Network{
		ProductId: "test",
		CodecId:   "script_codec",
		Script: `
function OnConnect(context) {
  console.log(JSON.stringify(context))
}
function Decode(context) {
  console.log("122")
  console.log(JSON.stringify(context))
}
function Encode(context) {
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
	c := codec.NewCodec(network)
	c.OnConnect(&codec.MockContext{DeviceId: "fff"})
	c.Decode(&codec.MockContext{DeviceId: "fff"})
	c.Encode(&codec.MockContext{DeviceId: "fff"})
	switch m := c.(type) {
	case codec.DeviceLifecycle:
		m.OnCreate(&codec.MockContext{DeviceId: "2222"})
	default:
	}
}