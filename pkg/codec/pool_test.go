package codec_test

import (
	"sync"
	"testing"
	"time"

	"go-iot/pkg/codec"
)

// 回归：脚本重新部署（NewScriptCodec → oldCodec.pool.Close()）与在途 FuncInvoke
// （已 Get 未 Put）并发时，旧代码会 panic: send on closed channel（进程崩溃）。
// 修复后：Put 对已关闭池丢弃 vm；Get 返回 nil；Close 幂等。
func TestVmPoolCloseConcurrent(t *testing.T) {
	p, err := codec.NewVmPool(`function OnConnect(){ return 1 }`, 20)
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	stop := make(chan struct{})
	// 模拟多个并发在途调用：Get →（模拟脚本执行）→ Put
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
				}
				vm := p.Get()
				if vm != nil {
					time.Sleep(time.Microsecond) // 模拟脚本执行窗口，放大竞争
					p.Put(vm)
				}
			}
		}()
	}
	// 模拟脚本重新部署触发 Close（可多次）
	wg.Add(1)
	go func() {
		defer wg.Done()
		time.Sleep(time.Millisecond)
		p.Close()
		p.Close() // 幂等
		close(stop)
	}()
	wg.Wait()

	// 关闭后收尾调用仍不应 panic
	vm := p.Get()
	if vm != nil {
		p.Put(vm)
	}
}
