package core

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// 离线命令队列：并发 Enqueue（API 下发）与 TakeAll（设备上线）不得崩溃
func TestOfflineCommandQueueConcurrent(t *testing.T) {
	q := NewMemoryOfflineCommandQueue()

	var stop atomic.Bool
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			for !stop.Load() {
				q.Enqueue(FuncInvoke{DeviceId: "d1", FunctionId: "f1", TraceId: "t"})
			}
		}(i)
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for !stop.Load() {
			q.TakeAll("d1")
		}
	}()
	time.Sleep(2 * time.Second)
	stop.Store(true)
	wg.Wait()
}

// 命令超时后迟到应答：由 TestDoCmdInvokeLateReplyNotBlocking 端到端覆盖
