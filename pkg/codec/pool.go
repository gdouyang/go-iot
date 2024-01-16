package codec

import (
	"errors"
	"fmt"
	"go-iot/pkg/eventbus"
	logs "go-iot/pkg/logger"

	"github.com/dop251/goja"
)

// javascript vm pool
type VmPool struct {
	chVM      chan *goja.Runtime
	productId string
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
			if p.productId != "" {
				PublishDebugMsg(p.productId, "", fmt.Sprintf("%v", v...))
			}
		})
		vm.Set("console", console)
		vm.Set("globe", &globe{vm: vm})
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

// 发布debug消息给事件总线
func PublishDebugMsg(productId, deviceId, data string) {
	if deviceId == "" {
		deviceId = "null"
	}
	eventbus.PublishDebug(eventbus.NewDebugMessage(deviceId, productId, data))
}
