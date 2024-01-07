package codec

import (
	"errors"
	logs "go-iot/pkg/logger"

	"github.com/dop251/goja"
)

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
		vm.Set("globe", &globe{vm: vm})
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

func (p *VmPool) Close() {
	close(p.chVM)
}
