package mqtt5

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go-iot/pkg/core"

	"github.com/stretchr/testify/require"
)

// 验证：platform ClientAndSession.GetConInfo() 并发调用（旧实现每次写共享
// s.connectInfo map，API 多请求并发查询连接信息会触发 fatal: concurrent map read
// and map write）。修复：GetConInfo 持 s.Lock 写 s.connectInfo。
func TestPlatform_GetConInfoConcurrentSafe(t *testing.T) {
	productId := fmt.Sprintf("pfci-%d", time.Now().UnixNano()%100000)
	deviceId := "pfci-0"
	port := startPlatformBroker(t, productId, deviceId)

	cp, cancel := platformConnect(t, port, deviceId)
	defer cancel()
	_ = cp
	time.Sleep(300 * time.Millisecond)

	sess := core.GetSession(deviceId)
	require.NotNil(t, sess, "device should be online")

	var stop atomic.Bool
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for !stop.Load() {
				_ = sess.GetConInfo()
			}
		}()
	}
	time.Sleep(1 * time.Second)
	stop.Store(true)
	wg.Wait()
}
