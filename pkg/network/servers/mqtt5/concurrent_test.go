package mqtt5_test

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go-iot/pkg/core"

	"github.com/stretchr/testify/require"
)

// 验证：ClientAndSession.GetConInfo() 并发调用（旧实现每次写共享 s.connectInfo
// map，在 API 多请求并发查询连接信息时会触发 fatal: concurrent map read and map
// write）。修复：GetConInfo 持 s.Lock 写 s.connectInfo。
func TestMqtt5_GetConInfoConcurrentSafe(t *testing.T) {
	productId := fmt.Sprintf("m5-ci-%d", time.Now().UnixNano()%100000)
	deviceId := "m5-ci-0"
	script := `
function OnConnect(context) {
  context.DeviceOnline(context.GetClientId())
}
function OnMessage(context) {}
`
	port, _, cleanup := startMqtt5MultiDevice(t, productId, script, deviceId)
	defer cleanup()

	_, cancel := mqtt5ConnectSilent(t, port, deviceId)
	defer cancel()
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
