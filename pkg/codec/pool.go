package codec

import (
	"errors"
	"fmt"
	"sync"

	"go-iot/pkg/core"
	logs "go-iot/pkg/logger"

	"github.com/dop251/goja"
)

var runtimeGlobes sync.Map // *goja.Runtime -> *globe

func bindRuntimeGlobe(vm *goja.Runtime, g *globe) {
	if vm == nil || g == nil {
		return
	}
	runtimeGlobes.Store(vm, g)
}

func dropRuntimeGlobe(vm *goja.Runtime) {
	if vm == nil {
		return
	}
	runtimeGlobes.Delete(vm)
}

// globeOf 按 VM 取创建时绑定的 globe（与 console.log 闭包同一指针）。
func globeOf(vm *goja.Runtime) *globe {
	if vm == nil {
		return nil
	}
	if v, ok := runtimeGlobes.Load(vm); ok {
		if g, ok := v.(*globe); ok {
			return g
		}
	}
	return nil
}

// javascript vm pool
type VmPool struct {
	chVM      chan *goja.Runtime
	productId string

	// mu + closed 保护 Close/Put 对 chVM 的竞争：
	// 脚本重新部署（NewScriptCodec）会关闭旧池，而并发在途的 FuncInvoke
	// （已 Get 未 Put）若向已 close 的 channel 发送会 panic: send on closed channel
	mu     sync.RWMutex
	closed bool
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
		g := &globe{vm: vm, productId: productId}
		console := vm.NewObject()
		for _, name := range []string{"log", "debug", "info", "warn", "error"} {
			level := name
			if name == "log" {
				level = "debug"
			}
			console.Set(name, func(v ...interface{}) {
				logs.Debugf("%v", v...)
				if p.productId != "" {
					core.DebugLog(level, g.currentDeviceId(), p.productId, fmt.Sprintf("%v", v...))
				}
			})
		}
		vm.Set("console", console)
		vm.Set("globe", g)
		bindRuntimeGlobe(vm, g)
		p.Put(vm)
	}
	return &p, nil
}

func (p *VmPool) SetProductId(productId string) {
	p.productId = productId
}

func (p *VmPool) Get() *goja.Runtime {
	// 只做快速检查后立即释放锁，接收本身不阻塞持锁（否则 Close 拿不到写锁形成死锁）；
	// 检查通过后 Close 并发发生是安全竞态：从已关闭 channel 接收不会 panic
	// （缓冲取空后返回 nil，由调用方判空处理）
	p.mu.RLock()
	closed := p.closed
	p.mu.RUnlock()
	if closed {
		return nil
	}
	return <-p.chVM
}

func (p *VmPool) Put(vm *goja.Runtime) {
	// 持 RLock 完成检查 + 发送：与 Close 的写锁互斥，保证发送时 channel 未被关闭。
	// channel 满时 send 阻塞期间持 RLock，但 Get 接收不持锁，消费不会停止，无死锁。
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.closed {
		// 池已关闭，丢弃 vm（交给 GC），避免 send on closed channel panic
		dropRuntimeGlobe(vm)
		return
	}
	p.chVM <- vm
}

func (p *VmPool) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		// 幂等：重复 Close 不 panic（恢复/重启流程可能多次触发）
		return
	}
	p.closed = true
	close(p.chVM)
	for vm := range p.chVM {
		dropRuntimeGlobe(vm)
	}
}
