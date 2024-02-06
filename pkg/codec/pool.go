package codec

import (
	"errors"
	"fmt"
	"go-iot/pkg/core"
	logs "go-iot/pkg/logger"

	"github.com/dop251/goja"
)

// javascript vm pool
type VmPool struct {
	chVM      chan *goja.Runtime
	productId string
}

// 创建js引擎池
func NewVmPool(src string, size int) (*VmPool, error) {
	return NewVmPool1(src, size, "")
}

// 创建js引擎池
func NewVmPool1(src string, size int, productId string) (*VmPool, error) {
	if len(src) == 0 {
		return nil, errors.New("script must be present")
	}
	program, err := goja.Compile("", src, false)
	if err != nil {
		return nil, err
	}
	p := VmPool{chVM: make(chan *goja.Runtime, size), productId: productId}
	for i := 0; i < size; i++ {
		vm := goja.New()
		_, err := vm.RunProgram(program)
		if err != nil {
			return nil, err
		}
		console := vm.NewObject()
		console.Set("log", func(v ...interface{}) {
			logs.Debugf("%v", v...)
			if p.productId != "" {
				core.DebugLog("", p.productId, fmt.Sprintf("%v", v...))
			}
		})
		vm.Set("console", console)
		vm.Set("globe", &globe{vm: vm, productId: productId})
		p.Put(vm)
	}
	return &p, nil
}

func (p *VmPool) SetProductId(productId string) {
	p.productId = productId
}

func (p *VmPool) Get() *goja.Runtime {
	vm := <-p.chVM
	return vm
}

func (p *VmPool) Put(vm *goja.Runtime) {
	p.chVM <- vm
}

func (p *VmPool) Close() {
	close(p.chVM)
}
