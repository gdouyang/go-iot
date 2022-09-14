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
