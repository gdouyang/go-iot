package mqtt5_test

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/url"
	"sync"
	"testing"
	"time"

	"go-iot/pkg/core"
	"go-iot/pkg/network"
	mqttserver "go-iot/pkg/network/servers/mqtt5"
	"go-iot/pkg/network/testhelper"

	"github.com/eclipse/paho.golang/autopaho"
	"github.com/eclipse/paho.golang/paho"
	"github.com/stretchr/testify/require"
)

// mqtt5ConnectSilent 同 mqtt5Connect，但 OnConnectError 不写测试日志：
// 供 Stop/断开场景使用（服务端关闭连接后 autopaho 会反复重连失败，
// 若回调写 t.Logf，测试结束后调用会 panic）
func mqtt5ConnectSilent(t *testing.T, port int32, clientID string) (*autopaho.ConnectionManager, context.CancelFunc) {
	t.Helper()
	u, err := url.Parse(fmt.Sprintf("mqtt://127.0.0.1:%d", port))
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	cliCfg := autopaho.ClientConfig{
		ServerUrls:                    []*url.URL{u},
		KeepAlive:                     20,
		CleanStartOnInitialConnection: true,
		ClientConfig: paho.ClientConfig{
			ClientID: clientID,
		},
	}
	c, err := autopaho.NewConnection(ctx, cliCfg)
	require.NoError(t, err)
	require.NoError(t, c.AwaitConnection(ctx))
	return c, cancel
}

// TestMqtt5_StopWithActiveConnections 覆盖 Stop 死锁场景：
// mochi server.Close 的 CloseAll 等待 Serve goroutine 退出，而它可能正阻塞在
// OnDisconnect 的 broker 锁上（同 clientID 重复连接路径）；修复前此用例挂起。
// startMqtt5MultiDevice 同 startMqtt5，但支持注册多个设备：
// 注意不能循环调 testhelper.SetupProductDevice（每次都会 InitCore 重置 DeviceStore，
// 前面注册的设备会被清空），需 InitCore 一次后批量 PutDevice
func startMqtt5MultiDevice(t *testing.T, productId, script string, deviceIds ...string) (int32, *mqttserver.Broker, func()) {
	t.Helper()
	port := testhelper.FreeTCPPort(t)
	testhelper.InitCore()
	core.DeleteProduct(productId)
	product, err := core.NewProduct(productId, map[string]string{}, core.TIME_SERISE_MOCK, testhelper.DefaultTSL())
	require.NoError(t, err)
	require.NoError(t, core.PutProduct(product))
	for _, d := range deviceIds {
		core.DeleteDevice(d)
		require.NoError(t, core.PutDevice(core.NewDevice(d, productId, 0)))
	}
	conf := network.NetworkConf{
		ProductId:     productId,
		CodecId:       "script_codec",
		Port:          port,
		Script:        script,
		Configuration: `{"host":"127.0.0.1","useTLS":false}`,
	}
	b := mqttserver.NewServer()
	require.NoError(t, b.Start(conf))
	_, err = core.NewCodec(conf.CodecId, conf.ProductId, conf.Script)
	require.NoError(t, err)
	time.Sleep(150 * time.Millisecond)
	return port, b, func() { _ = b.Stop() }
}

func TestMqtt5_StopWithActiveConnections(t *testing.T) {
	productId := fmt.Sprintf("m5-stop-%d", time.Now().UnixNano()%100000)
	script := `
function OnConnect(context) {
  context.DeviceOnline(context.GetClientId())
}
function OnMessage(context) {}
`
	// 注册 5 个设备并连接，连接建立后各有 session
	deviceIds := make([]string, 0, 5)
	for i := 0; i < 5; i++ {
		deviceIds = append(deviceIds, fmt.Sprintf("m5-stop-%d", i))
	}
	port, b, cleanup := startMqtt5MultiDevice(t, productId, script, deviceIds...)
	cancels := make([]context.CancelFunc, 0, 5)
	for _, did := range deviceIds {
		_, cancel := mqtt5ConnectSilent(t, port, did)
		cancels = append(cancels, cancel)
	}
	defer func() {
		for _, cancel := range cancels {
			cancel()
		}
	}()
	time.Sleep(300 * time.Millisecond)
	require.EqualValues(t, 5, b.TotalConnection())
	for _, did := range deviceIds {
		assertSession(t, did, true)
	}

	done := make(chan struct{})
	go func() {
		cleanup()
		close(done)
	}()
	select {
	case <-done:
		// Stop 正常返回
	case <-time.After(15 * time.Second):
		t.Fatal("broker Stop deadlocked with active connections")
	}
	require.EqualValues(t, 0, b.TotalConnection())
	// Stop 后 session 全部下线
	for _, did := range deviceIds {
		waitSessionGone(t, did)
	}
}

// TestMqtt5_DuplicateClientID_KickOld 覆盖同 clientID 重复连接路径：
// mochi attachClient 会同步触发旧连接的 OnDisconnect（原锁重入死锁点，
// 修复前第二个连接建立即死锁挂起）。
func TestMqtt5_DuplicateClientID_KickOld(t *testing.T) {
	productId := fmt.Sprintf("m5-dup-%d", time.Now().UnixNano()%100000)
	script := `function OnConnect(context) {} function OnMessage(context) {}`
	port, b, cleanup := startMqtt5(t, productId, script)
	defer cleanup()

	_, cancel1 := mqtt5Connect(t, port, "dup-id")
	defer cancel1()
	time.Sleep(200 * time.Millisecond)
	require.EqualValues(t, 1, b.TotalConnection())

	// 同 ID 第二个连接：若 OnDisconnect 死锁，AwaitConnection 10s 超时
	_, cancel2 := mqtt5Connect(t, port, "dup-id")
	defer cancel2()

	// c1 被踢后 autopaho 会自动重连（同 ID），与 c2 互相踢，连接数动态波动；
	// 核心断言：无死锁无 panic，且能稳定恢复在线（修复前此路径死锁挂起）
	deadline := time.Now().Add(10 * time.Second)
	for {
		n := b.TotalConnection()
		if n >= 1 {
			require.LessOrEqual(t, n, int32(2), "unexpected connection count %d", n)
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("duplicate clientID connections not recovered, TotalConnection=%d", n)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// assertSession 仅比较 nil 且错误信息只包含 deviceId 字符串：
// 避免断言失败时 testify 对包含 sync.Mutex 的 session 值执行 %#v 格式化，
// 该格式化会并发读取锁状态，与持锁写入的 goroutine 构成数据竞争。
func assertSession(t *testing.T, deviceId string, wantOnline bool) {
	t.Helper()
	if got := core.GetSession(deviceId) != nil; got != wantOnline {
		t.Fatalf("session %s online=%v, want %v", deviceId, got, wantOnline)
	}
}

// waitSessionGone 轮询等待 session 下线：OnDisconnect 的清理过程是异步的
// （broker 锁 TryLock 重试与 DelSessionWithTimeoutCheck），TotalConnection()==0
// 不保证 session 已删除，一次性断言可能落入清理窗口。
func waitSessionGone(t *testing.T, deviceId string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		if core.GetSession(deviceId) == nil {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("session %s not cleaned up within 5s", deviceId)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// rawMqtt5Connect 手动构造 MQTT5 CONNECT 包并返回原始 TCP 连接：
// 用于模拟异常断开（不发 DISCONNECT 直接 Close）
func rawMqtt5Connect(t *testing.T, port int32, clientID string) net.Conn {
	t.Helper()
	conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	require.NoError(t, err)

	var vh []byte
	vh = append(vh, 0x00, 0x04, 'M', 'Q', 'T', 'T') // protocol name "MQTT"
	vh = append(vh, 0x05)                          // protocol level 5
	vh = append(vh, 0x02)                          // connect flags: clean start
	vh = append(vh, 0x00, 0x1E)                    // keep alive 30s
	vh = append(vh, 0x00)                          // properties length 0
	cid := []byte(clientID)
	vh = append(vh, byte(len(cid)>>8), byte(len(cid)))
	vh = append(vh, cid...)
	_, err = conn.Write(append([]byte{0x10, byte(len(vh))}, vh...))
	require.NoError(t, err)

	// 读 CONNACK：0x20 + remaining length(2) + session present + reason code + properties(0)
	header := make([]byte, 2)
	_, err = io.ReadFull(conn, header)
	require.NoError(t, err)
	require.Equal(t, byte(0x20), header[0], "unexpected first byte %#x", header[0])
	body := make([]byte, header[1])
	_, err = io.ReadFull(conn, body)
	require.NoError(t, err)
	require.Equal(t, byte(0), body[1], "connect refused, reason=%d", body[1])
	return conn
}

// TestMqtt5_AbruptDisconnect 覆盖异常断开（TCP 直接关闭、不发 DISCONNECT）：
// broker 通过读 EOF 触发 OnDisconnect(err)，必须完成 map 删除 + session 下线
func TestMqtt5_AbruptDisconnect(t *testing.T) {
	productId := fmt.Sprintf("m5-abrupt-%d", time.Now().UnixNano()%100000)
	script := `
function OnConnect(context) {
  context.DeviceOnline(context.GetClientId())
}
function OnMessage(context) {}
`
	port, b, cleanup := startMqtt5(t, productId, script)
	defer cleanup()

	conn := rawMqtt5Connect(t, port, "dev-m5-ok")
	time.Sleep(300 * time.Millisecond)
	require.EqualValues(t, 1, b.TotalConnection())
	assertSession(t, "dev-m5-ok", true)

	// 异常断开：直接关闭 TCP，不发 DISCONNECT
	require.NoError(t, conn.Close())
	deadline := time.Now().Add(5 * time.Second)
	for {
		if b.TotalConnection() == 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("abrupt disconnect not cleaned up, TotalConnection=%d", b.TotalConnection())
		}
		time.Sleep(50 * time.Millisecond)
	}
	waitSessionGone(t, "dev-m5-ok")
}
// 原 TryLock 失败跳过版本会残留 session（TotalConnection 不归零、GetSession 仍在）；
// 修复后 OnDisconnect 必须完成 map 删除 + session 下线。
// TestMqtt5_DisconnectCleanup 覆盖设备主动断开（优雅 Disconnect）后的清理：
// 原 TryLock 失败跳过版本会残留 session（TotalConnection 不归零、GetSession 仍在）；
// 修复后 OnDisconnect 必须完成 map 删除 + session 下线。
func TestMqtt5_DisconnectCleanup(t *testing.T) {
	productId := fmt.Sprintf("m5-disc-%d", time.Now().UnixNano()%100000)
	script := `
function OnConnect(context) {
  context.DeviceOnline(context.GetClientId())
}
function OnMessage(context) {}
`
	port, b, cleanup := startMqtt5(t, productId, script)
	defer cleanup()

	c, cancel := mqtt5Connect(t, port, "dev-m5-ok")
	defer cancel()
	time.Sleep(300 * time.Millisecond)
	require.EqualValues(t, 1, b.TotalConnection())
	assertSession(t, "dev-m5-ok", true)

	require.NoError(t, c.Disconnect(context.Background()))
	// 等待 OnDisconnect 完成清理
	deadline := time.Now().Add(5 * time.Second)
	for {
		if b.TotalConnection() == 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("client disconnect not cleaned up, TotalConnection=%d", b.TotalConnection())
		}
		time.Sleep(50 * time.Millisecond)
	}
	waitSessionGone(t, "dev-m5-ok")
}

// TestMqtt5_StopThenLateConnection 覆盖 Stop 后迟到的连接（autopaho 自动重连）：
// Stop 把 clients 置空 map（非 nil），迟到的 OnConnectAuthenticate 执行
// clients[id]=client 不得 panic（旧版置 nil 会 "assignment to entry in nil map" 崩溃）
func TestMqtt5_StopThenLateConnection(t *testing.T) {
	productId := fmt.Sprintf("m5-late-%d", time.Now().UnixNano()%100000)
	script := `function OnConnect(context) {} function OnMessage(context) {}`
	port, b, cleanup := startMqtt5(t, productId, script)

	// autopaho 连接（默认自动重连：Stop 后会持续重试）
	_, cancel := mqtt5ConnectSilent(t, port, "late-id")
	defer cancel()
	time.Sleep(200 * time.Millisecond)
	require.EqualValues(t, 1, b.TotalConnection())

	// Stop：clients 置空 map
	cleanup()
	require.EqualValues(t, 0, b.TotalConnection())

	// 等待 autopaho 重连尝试触发迟到 auth——不 panic 即通过
	time.Sleep(1500 * time.Millisecond)
	require.EqualValues(t, 0, b.TotalConnection())
}

// TestMqtt5_PublishDuringDuplicateStorm 覆盖同 ID 抢占风暴期间的并发消息发布：
// OnPublished 并发读 clients map（RLock 守卫），风暴 + 发布同时进行不得 panic/死锁
// （修复前 OnPublished 无锁读会 fatal: concurrent map read and map write）
func TestMqtt5_PublishDuringDuplicateStorm(t *testing.T) {
	productId := fmt.Sprintf("m5-storm-%d", time.Now().UnixNano()%100000)
	script := `
function OnConnect(context) {
  context.DeviceOnline(context.GetClientId())
}
function OnMessage(context) {}
`
	port, b, cleanup := startMqtt5MultiDevice(t, productId, script, "storm-id")
	defer cleanup()

	// 两个客户端同 ID 互踢（风暴）
	c1, cancel1 := mqtt5Connect(t, port, "storm-id")
	defer cancel1()
	time.Sleep(200 * time.Millisecond)
	c2, cancel2 := mqtt5Connect(t, port, "storm-id")
	defer cancel2()

	// 风暴期间双方持续发布 QoS0 消息：触发 OnPublished 并发 map 读
	stop := make(chan struct{})
	var wg sync.WaitGroup
	for _, cm := range []*autopaho.ConnectionManager{c1, c2} {
		wg.Add(1)
		go func(cm *autopaho.ConnectionManager) {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
				}
				// 连接可能被踢/重连中，发布失败可忽略——断言核心是不 panic 不死锁
				_, _ = cm.Publish(context.Background(), &paho.Publish{Topic: "dev/prop", QoS: 0, Payload: []byte(`{"t":1}`)})
			}
		}(cm)
	}
	time.Sleep(3 * time.Second)
	close(stop)
	wg.Wait()

	// 风暴后系统仍存活：新连接可正常建立
	_, cancel3 := mqtt5ConnectSilent(t, port, "storm-id")
	defer cancel3()
	time.Sleep(200 * time.Millisecond)
	require.GreaterOrEqual(t, b.TotalConnection(), int32(1))
}
