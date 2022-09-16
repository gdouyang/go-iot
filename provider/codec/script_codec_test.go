package codec

import (
	"fmt"
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
	var network Network = Network{
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
	codec := NewCodec(network)
	codec.OnConnect(&MockContext{DeviceId: "fff"})
	codec.Decode(&MockContext{DeviceId: "fff"})
	codec.Encode(&MockContext{DeviceId: "fff"})
	switch m := codec.(type) {
	case DeviceLifecycle:
		m.OnCreate(&MockContext{DeviceId: "2222"})
	default:
	}
}
