package websocketserver_test

import (
	"fmt"
	"sync"
	"testing"
	"time"

	_ "go-iot/pkg/codec"
	"go-iot/pkg/core"
	"go-iot/pkg/network"
	websocketserver "go-iot/pkg/network/servers/websocket"
	"go-iot/pkg/network/testhelper"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

// startWsServerMultiDevice 启动 websocket 服务并批量注册设备：
// 注意不能循环调 testhelper.SetupProductDevice（每次都会 InitCore 重置 DeviceStore，
// 前面注册的设备会被清空），需 InitCore 一次后批量 PutDevice
func startWsServerMultiDevice(t *testing.T, productId string, deviceIds ...string) (int32, *websocketserver.WebSocketServer, func()) {
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
	script := `
function OnConnect(context) {
	var deviceId = context.GetQuery("deviceId")
	context.DeviceOnline(deviceId)
}
function OnMessage(context) {
	context.GetSession().SendText(JSON.stringify({ok:true}))
}
`
	conf := network.NetworkConf{
		ProductId:     productId,
		CodecId:       "script_codec",
		Port:          port,
		Script:        script,
		Configuration: `{"host":"127.0.0.1","useTLS":false,"routers":[{"url":"/socket"}]}`,
	}
	srv := websocketserver.NewServer()
	require.NoError(t, srv.Start(conf))
	_, err = core.NewCodec(conf.CodecId, conf.ProductId, conf.Script)
	require.NoError(t, err)
	time.Sleep(50 * time.Millisecond)
	return port, srv, func() { _ = srv.Stop() }
}

// wsDial 连接 websocket（deviceId 通过 query 参数传入，OnConnect 脚本据此上线）
func wsDial(t *testing.T, port int32, deviceId string) *websocket.Conn {
	t.Helper()
	u := fmt.Sprintf("ws://127.0.0.1:%d/socket?deviceId=%s", port, deviceId)
	conn, _, err := websocket.DefaultDialer.Dial(u, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

func wsWaitSessionNil(t *testing.T, deviceId string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		if core.GetSession(deviceId) == nil {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("session %s not cleaned up", deviceId)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func wsWaitTotal(t *testing.T, srv *websocketserver.WebSocketServer, want int32) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		if srv.TotalConnection() == want {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("TotalConnection=%d, want %d", srv.TotalConnection(), want)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// TestWebSocketServer_StopWithActiveConnections 覆盖带活动连接的 Stop：
// 必须正常返回（不挂起）、连接全部关闭、session 全部下线
func TestWebSocketServer_StopWithActiveConnections(t *testing.T) {
	productId := fmt.Sprintf("ws-stop-%d", time.Now().UnixNano()%100000)
	deviceIds := []string{"ws-stop-0", "ws-stop-1", "ws-stop-2"}
	port, srv, cleanup := startWsServerMultiDevice(t, productId, deviceIds...)

	conns := make([]*websocket.Conn, 0, len(deviceIds))
	for _, did := range deviceIds {
		conns = append(conns, wsDial(t, port, did))
	}
	defer func() {
		for _, c := range conns {
			_ = c.Close()
		}
	}()
	time.Sleep(300 * time.Millisecond)
	require.EqualValues(t, len(deviceIds), srv.TotalConnection())
	for _, did := range deviceIds {
		require.NotNil(t, core.GetSession(did))
	}

	done := make(chan struct{})
	go func() {
		cleanup() // srv.Stop()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(15 * time.Second):
		t.Fatal("websocket server Stop deadlocked with active connections")
	}
	// Stop 后连接全部关闭、map 清空、session 全下线
	wsWaitTotal(t, srv, 0)
	for _, did := range deviceIds {
		require.Nil(t, core.GetSession(did))
	}
}

// TestWebSocketServer_AbruptDisconnect 覆盖异常断开（TCP 直接关闭、不发 Close 帧）：
// readLoop 读错误退出 → close1 必须完成 map 删除 + session 下线
func TestWebSocketServer_AbruptDisconnect(t *testing.T) {
	productId := fmt.Sprintf("ws-abrupt-%d", time.Now().UnixNano()%100000)
	port, srv, cleanup := startWsServerMultiDevice(t, productId, "ws-abrupt-0")
	defer cleanup()

	conn := wsDial(t, port, "ws-abrupt-0")
	time.Sleep(300 * time.Millisecond)
	require.EqualValues(t, 1, srv.TotalConnection())
	require.NotNil(t, core.GetSession("ws-abrupt-0"))

	// 异常断开：直接关闭 TCP，不发 Close 帧
	require.NoError(t, conn.Close())
	wsWaitTotal(t, srv, 0)
	// 清理异步（close1：先 removeClient 后 DelSession），轮询等待 session 归零
	wsWaitSessionNil(t, "ws-abrupt-0")
}

// TestWebSocketServer_DisconnectCleanup 覆盖优雅断开（客户端发 Close 帧）后的清理：
// 必须完成 map 删除 + session 下线
func TestWebSocketServer_DisconnectCleanup(t *testing.T) {
	productId := fmt.Sprintf("ws-disc-%d", time.Now().UnixNano()%100000)
	port, srv, cleanup := startWsServerMultiDevice(t, productId, "ws-disc-0")
	defer cleanup()

	conn := wsDial(t, port, "ws-disc-0")
	time.Sleep(300 * time.Millisecond)
	require.EqualValues(t, 1, srv.TotalConnection())
	require.NotNil(t, core.GetSession("ws-disc-0"))

	// 优雅断开：发送 Close 帧
	require.NoError(t, conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseGoingAway, "")))
	wsWaitTotal(t, srv, 0)
	wsWaitSessionNil(t, "ws-disc-0")
}

// TestWebSocketServer_DuplicateDeviceId 覆盖同 deviceId 重复连接：
// 旧连接断开后不得误删新连接的 session（对应 mqtt5 同 ID 抢占误伤的修复验证；
// core session 按 deviceId 单例，旧连接 close1 的 DelSession 必须不能误伤新连接）
func TestWebSocketServer_DuplicateDeviceId(t *testing.T) {
	productId := fmt.Sprintf("ws-dup-%d", time.Now().UnixNano()%100000)
	port, srv, cleanup := startWsServerMultiDevice(t, productId, "ws-dup-0")
	defer cleanup()

	conn1 := wsDial(t, port, "ws-dup-0")
	time.Sleep(200 * time.Millisecond)
	require.EqualValues(t, 1, srv.TotalConnection())
	require.NotNil(t, core.GetSession("ws-dup-0"))

	// 同 deviceId 第二个连接
	conn2 := wsDial(t, port, "ws-dup-0")
	time.Sleep(200 * time.Millisecond)
	require.EqualValues(t, 2, srv.TotalConnection())
	require.NotNil(t, core.GetSession("ws-dup-0"))

	// 断开旧连接：新连接的连接数与 session 必须保留
	require.NoError(t, conn1.Close())
	wsWaitTotal(t, srv, 1)
	require.EqualValues(t, 1, srv.TotalConnection())
	// 等旧连接 close1 清理完成（DelSessionWithTimeoutCheck 是异步竞态点）
	time.Sleep(500 * time.Millisecond)
	require.NotNil(t, core.GetSession("ws-dup-0"), "new connection session must survive old disconnect")

	// 新连接仍可正常收发
	require.NoError(t, conn2.WriteMessage(websocket.TextMessage, []byte(`{"temperature":1}`)))
	_ = conn2.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, resp, err := conn2.ReadMessage()
	require.NoError(t, err)
	require.Contains(t, string(resp), "ok")
}

// TestWebSocketServer_SendAfterDisconnect 覆盖断开后 SendText 的 panic 风险：
// Close 已 close(send) 通道，此后任何 SendText 都会 "send on closed channel" panic
// （生产场景：设备断开瞬间 OnMessage 脚本响应消息 → 进程崩溃）
func TestWebSocketServer_SendAfterDisconnect(t *testing.T) {
	productId := fmt.Sprintf("ws-send-%d", time.Now().UnixNano()%100000)
	port, srv, cleanup := startWsServerMultiDevice(t, productId, "ws-send-0")
	defer cleanup()

	conn := wsDial(t, port, "ws-send-0")
	time.Sleep(300 * time.Millisecond)
	require.EqualValues(t, 1, srv.TotalConnection())

	sess := core.GetSession("ws-send-0")
	require.NotNil(t, sess)
	wsSess, ok := sess.(*websocketserver.WebsocketSession)
	require.True(t, ok, "session should be *WebsocketSession")

	// 服务端主动断开（模拟 Stop/掉线清理路径：Close → close(s.send)）
	require.NoError(t, wsSess.Disconnect())
	wsWaitTotal(t, srv, 0)

	// 断开后 SendText 不得 panic（修复前：send on closed channel）
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, _, _ = conn.ReadMessage() // 消费服务端 Close 帧，避免影响后续
	require.Error(t, wsSess.SendText("after-disconnect"))
}

// TestWebSocketServer_ConcurrentConnections 覆盖并发连接场景：
// session id 必须全局唯一（修复前用 time.Now().UnixNano()，Windows 时钟精度 ~ms 级
// 并发连接 id 几乎必撞 → clients map 覆盖 → TotalConnection 丢失）
func TestWebSocketServer_ConcurrentConnections(t *testing.T) {
	productId := fmt.Sprintf("ws-conc-%d", time.Now().UnixNano()%100000)
	deviceIds := make([]string, 0, 20)
	for i := 0; i < 20; i++ {
		deviceIds = append(deviceIds, fmt.Sprintf("ws-conc-%d", i))
	}
	port, srv, cleanup := startWsServerMultiDevice(t, productId, deviceIds...)
	defer cleanup()

	// 20 个连接并发建立
	var wg sync.WaitGroup
	conns := make([]*websocket.Conn, 0, 20)
	var mu sync.Mutex
	for _, did := range deviceIds {
		wg.Add(1)
		go func(did string) {
			defer wg.Done()
			u := fmt.Sprintf("ws://127.0.0.1:%d/socket?deviceId=%s", port, did)
			conn, _, err := websocket.DefaultDialer.Dial(u, nil)
			if err != nil {
				t.Errorf("dial %s: %v", did, err)
				return
			}
			mu.Lock()
			conns = append(conns, conn)
			mu.Unlock()
		}(did)
	}
	wg.Wait()
	defer func() {
		for _, c := range conns {
			_ = c.Close()
		}
	}()

	// 全部连接必须在 map 中（修复前 id 碰撞会丢失连接）；
	// DeviceOnline 在 readLoop 异步执行，等待所有 session 就绪
	wsWaitTotal(t, srv, int32(len(deviceIds)))
	deadline := time.Now().Add(5 * time.Second)
	for {
		all := true
		for _, did := range deviceIds {
			if core.GetSession(did) == nil {
				all = false
				break
			}
		}
		if all {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("not all sessions online")
		}
		time.Sleep(50 * time.Millisecond)
	}
}
