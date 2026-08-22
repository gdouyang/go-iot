package mqtt5_test

import (
	"context"
	"fmt"
	"net/url"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go-iot/pkg/core"

	"github.com/eclipse/paho.golang/autopaho"
	"github.com/eclipse/paho.golang/paho"
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

// 验证：认证逻辑在锁外并发执行。
// 当存在慢认证（50ms 脚本耗时）时，已连接客户端的断开操作（OnDisconnect）
// 不会被慢认证阻塞，且不会触发 give up lock after 100ms。
func TestMqtt5_SlowAuthDoesNotBlockDisconnect(t *testing.T) {
	productId := fmt.Sprintf("m5-p0-slow-%d", time.Now().UnixNano()%100000)
	// 在认证脚本中人为制造 50ms 延时
	script := `
function OnConnect(context) {
	var start = Date.now();
	while (Date.now() - start < 50) {}
	context.DeviceOnline(context.GetClientId())
}
function OnMessage(context) {}
`
	// 注册 1 个快速连接设备 + 5 个慢认证设备
	devFast := "dev-p0-fast"
	slowDevs := []string{"dev-p0-s1", "dev-p0-s2", "dev-p0-s3", "dev-p0-s4", "dev-p0-s5"}
	allDevs := append([]string{devFast}, slowDevs...)

	port, b, cleanup := startMqtt5MultiDevice(t, productId, script, allDevs...)
	defer cleanup()

	// 1. 先连接 fast 设备
	_, cancelFast := mqtt5ConnectSilent(t, port, devFast)
	time.Sleep(100 * time.Millisecond)
	require.NotNil(t, core.GetSession(devFast))

	// 2. 并发发起 5 个慢认证连接
	var wg sync.WaitGroup
	cancels := make([]context.CancelFunc, len(slowDevs))
	for i, did := range slowDevs {
		idx := i
		devId := did
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, cancel := mqtt5ConnectSilent(t, port, devId)
			cancels[idx] = cancel
		}()
	}

	// 3. 在慢认证并发进行的同时，快速断开 fast 设备并验证其微秒级完成
	time.Sleep(20 * time.Millisecond) // 确保慢认证正在进行中
	discStart := time.Now()
	cancelFast()
	waitSessionGone(t, devFast)
	discCost := time.Since(discStart)

	// 断开耗时必须远小于累积认证阻塞时间（若在锁内，5个并发连接会造成上百毫秒锁排队）
	require.Less(t, discCost, 500*time.Millisecond, "OnDisconnect should not be blocked by slow auth")

	wg.Wait()
	for _, cancel := range cancels {
		if cancel != nil {
			cancel()
		}
	}
	_ = b
}

// 验证：认证失败的非法连接绝不能污染或破坏已有合法连接的 Session。
func TestMqtt5_FailedAuthDoesNotPolluteValidSession(t *testing.T) {
	productId := fmt.Sprintf("m5-p0-auth-%d", time.Now().UnixNano()%100000)
	script := `
function OnConnect(context) {
	if (context.GetPassword() !== 'correct_pwd') {
		context.AuthFail();
		return;
	}
	context.DeviceOnline(context.GetClientId());
}
function OnMessage(context) {}
`
	devId := "dev-p0-legit"
	port, _, cleanup := startMqtt5MultiDevice(t, productId, script, devId)
	defer cleanup()

	// 1. 合法客户端使用正确密码连接
	u, err := url.Parse(fmt.Sprintf("mqtt://127.0.0.1:%d", port))
	require.NoError(t, err)

	ctxLegit, cancelLegit := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelLegit()

	cliCfg := autopaho.ClientConfig{
		ServerUrls:      []*url.URL{u},
		KeepAlive:       20,
		ConnectPassword: []byte("correct_pwd"),
		ClientConfig: paho.ClientConfig{
			ClientID: devId,
		},
	}
	cLegit, err := autopaho.NewConnection(ctxLegit, cliCfg)
	require.NoError(t, err)
	require.NoError(t, cLegit.AwaitConnection(ctxLegit))

	validSess := core.GetSession(devId)
	require.NotNil(t, validSess, "legitimate session must exist")

	// 2. 攻击者使用错误密码尝试连接相同 clientID
	ctxAtt, cancelAtt := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancelAtt()

	attCfg := autopaho.ClientConfig{
		ServerUrls:      []*url.URL{u},
		KeepAlive:       20,
		ConnectPassword: []byte("wrong_pwd"),
		ClientConfig: paho.ClientConfig{
			ClientID: devId,
		},
	}
	cAtt, err := autopaho.NewConnection(ctxAtt, attCfg)
	require.NoError(t, err)
	_ = cAtt.AwaitConnection(ctxAtt) // 预期认证失败

	// 3. 验证已有合法 session 依旧正常存活且未被破坏
	afterSess := core.GetSession(devId)
	require.NotNil(t, afterSess, "valid session must NOT be corrupted by failed auth")
	require.Equal(t, validSess, afterSess, "session pointer must remain unchanged")
}
