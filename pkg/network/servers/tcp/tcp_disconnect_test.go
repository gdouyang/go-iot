package tcpserver_test

import (
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	_ "go-iot/pkg/codec"
	"go-iot/pkg/core"
	"go-iot/pkg/network"
	tcpserver "go-iot/pkg/network/servers/tcp"
	"go-iot/pkg/network/testhelper"

	"github.com/stretchr/testify/require"
)

const tcpDisconnectScript = `
function OnConnect(context) {
  console.log("OnConnect")
}
function OnMessage(context) {
	var data = JSON.parse(context.MsgToString())
	context.DeviceOnline(data.deviceId)
}
function OnInvoke(context) {}
`

// startTcpServerMultiDevice 启动 tcp 服务并批量注册设备：
// 注意不能循环调 testhelper.SetupProductDevice（每次都会 InitCore 重置 DeviceStore）
func startTcpServerMultiDevice(t *testing.T, productId string, deviceIds ...string) (int32, *tcpserver.TcpServer, func()) {
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
		ProductId: productId,
		CodecId:   "script_codec",
		Port:      port,
		Script:    tcpDisconnectScript,
		Configuration: fmt.Sprintf(`{"host":"127.0.0.1","port":%d,"useTLS":false,
			"delimeter":{"type":"Delimited","delimited":"}"}}`, port),
	}
	srv := tcpserver.NewServer()
	require.NoError(t, srv.Start(conf))
	_, err = core.NewCodec(conf.CodecId, conf.ProductId, conf.Script)
	require.NoError(t, err)
	time.Sleep(50 * time.Millisecond)
	return port, srv, func() { _ = srv.Stop() }
}

// tcpConnect 连接 tcp 并发送上线消息（deviceId 在第一个 } 前闭合，分隔符按 } 拆包）
func tcpConnect(t *testing.T, port int32, deviceId string) net.Conn {
	t.Helper()
	conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })
	_, err = conn.Write([]byte(fmt.Sprintf(`{"deviceId":"%s"}`, deviceId)))
	require.NoError(t, err)
	return conn
}

func tcpWaitTotal(t *testing.T, srv *tcpserver.TcpServer, want int32) {
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

func tcpWaitSessionNil(t *testing.T, deviceId string) {
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

// TestTcpServer_StopWithActiveConnections 覆盖带活动连接的 Stop：
// 必须正常返回（不挂起）、连接全部关闭、session 全部下线
func TestTcpServer_StopWithActiveConnections(t *testing.T) {
	productId := fmt.Sprintf("tcp-stop-%d", time.Now().UnixNano()%100000)
	deviceIds := []string{"tcp-stop-0", "tcp-stop-1", "tcp-stop-2"}
	port, srv, cleanup := startTcpServerMultiDevice(t, productId, deviceIds...)

	conns := make([]net.Conn, 0, len(deviceIds))
	for _, did := range deviceIds {
		conns = append(conns, tcpConnect(t, port, did))
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
		t.Fatal("tcp server Stop deadlocked with active connections")
	}
	// Stop 后连接全部关闭、map 清空、session 全下线
	tcpWaitTotal(t, srv, 0)
	for _, did := range deviceIds {
		tcpWaitSessionNil(t, did)
	}
}

// TestTcpServer_AbruptDisconnect 覆盖异常断开（TCP 直接关闭）：
// readLoop 读错误退出 → close1 必须完成 map 删除 + session 下线
func TestTcpServer_AbruptDisconnect(t *testing.T) {
	productId := fmt.Sprintf("tcp-abrupt-%d", time.Now().UnixNano()%100000)
	port, srv, cleanup := startTcpServerMultiDevice(t, productId, "tcp-abrupt-0")
	defer cleanup()

	conn := tcpConnect(t, port, "tcp-abrupt-0")
	time.Sleep(300 * time.Millisecond)
	require.EqualValues(t, 1, srv.TotalConnection())
	require.NotNil(t, core.GetSession("tcp-abrupt-0"))

	// 异常断开：直接关闭 TCP
	require.NoError(t, conn.Close())
	tcpWaitTotal(t, srv, 0)
	tcpWaitSessionNil(t, "tcp-abrupt-0")
}

// TestTcpServer_DisconnectCleanup 覆盖服务端主动断开（session.Disconnect）后的清理
func TestTcpServer_DisconnectCleanup(t *testing.T) {
	productId := fmt.Sprintf("tcp-disc-%d", time.Now().UnixNano()%100000)
	port, srv, cleanup := startTcpServerMultiDevice(t, productId, "tcp-disc-0")
	defer cleanup()

	tcpConnect(t, port, "tcp-disc-0")
	time.Sleep(300 * time.Millisecond)
	require.EqualValues(t, 1, srv.TotalConnection())
	sess := core.GetSession("tcp-disc-0")
	require.NotNil(t, sess)
	tcpSess, ok := sess.(*tcpserver.TcpSession)
	require.True(t, ok, "session should be *TcpSession")

	require.NoError(t, tcpSess.Disconnect())
	tcpWaitTotal(t, srv, 0)
	tcpWaitSessionNil(t, "tcp-disc-0")
}

// TestTcpServer_DuplicateDeviceId 覆盖同 deviceId 重复连接：
// 旧连接断开后不得误删新连接的 session（core session 按 deviceId 单例）
func TestTcpServer_DuplicateDeviceId(t *testing.T) {
	productId := fmt.Sprintf("tcp-dup-%d", time.Now().UnixNano()%100000)
	port, srv, cleanup := startTcpServerMultiDevice(t, productId, "tcp-dup-0")
	defer cleanup()

	conn1 := tcpConnect(t, port, "tcp-dup-0")
	time.Sleep(200 * time.Millisecond)
	require.EqualValues(t, 1, srv.TotalConnection())
	require.NotNil(t, core.GetSession("tcp-dup-0"))

	// 同 deviceId 第二个连接
	_ = tcpConnect(t, port, "tcp-dup-0")
	time.Sleep(200 * time.Millisecond)
	require.EqualValues(t, 2, srv.TotalConnection())

	// 断开旧连接：新连接的连接数与 session 必须保留
	require.NoError(t, conn1.Close())
	tcpWaitTotal(t, srv, 1)
	require.EqualValues(t, 1, srv.TotalConnection())
	// 等旧连接 close1 清理完成（DelSessionWithTimeoutCheck 是异步竞态点）
	time.Sleep(500 * time.Millisecond)
	require.NotNil(t, core.GetSession("tcp-dup-0"), "new connection session must survive old disconnect")
}

// TestTcpServer_ConcurrentConnections 覆盖并发连接场景：
// session id 必须全局唯一（修复前用 time.Now().UnixNano()，Windows 时钟精度 ~ms 级
// 并发连接 id 几乎必撞 → clients map 覆盖 → TotalConnection 丢失）
func TestTcpServer_ConcurrentConnections(t *testing.T) {
	productId := fmt.Sprintf("tcp-conc-%d", time.Now().UnixNano()%100000)
	deviceIds := make([]string, 0, 20)
	for i := 0; i < 20; i++ {
		deviceIds = append(deviceIds, fmt.Sprintf("tcp-conc-%d", i))
	}
	port, srv, cleanup := startTcpServerMultiDevice(t, productId, deviceIds...)
	defer cleanup()

	var wg sync.WaitGroup
	conns := make([]net.Conn, 0, 20)
	var mu sync.Mutex
	for _, did := range deviceIds {
		wg.Add(1)
		go func(did string) {
			defer wg.Done()
			conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", port))
			if err != nil {
				t.Errorf("dial: %v", err)
				return
			}
			_, err = conn.Write([]byte(fmt.Sprintf(`{"deviceId":"%s"}`, did)))
			if err != nil {
				t.Errorf("write: %v", err)
				_ = conn.Close()
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

	// 全部连接必须在 map 中（修复前 id 碰撞会丢失连接）
	tcpWaitTotal(t, srv, int32(len(deviceIds)))
}

// TestTcpServer_SendAfterDisconnect 覆盖断开后 Send/SendHex 的 panic 风险：
// Close 已 close(send) 通道，此后任何 Send 都会 "send on closed channel" panic
// （生产场景：设备断开瞬间 OnMessage 脚本响应消息 → 进程崩溃）
func TestTcpServer_SendAfterDisconnect(t *testing.T) {
	productId := fmt.Sprintf("tcp-send-%d", time.Now().UnixNano()%100000)
	port, srv, cleanup := startTcpServerMultiDevice(t, productId, "tcp-send-0")
	defer cleanup()

	tcpConnect(t, port, "tcp-send-0")
	time.Sleep(300 * time.Millisecond)
	require.EqualValues(t, 1, srv.TotalConnection())

	sess := core.GetSession("tcp-send-0")
	require.NotNil(t, sess)
	tcpSess, ok := sess.(*tcpserver.TcpSession)
	require.True(t, ok, "session should be *TcpSession")

	// 服务端主动断开（Close → close(s.send)）
	require.NoError(t, tcpSess.Disconnect())
	tcpWaitTotal(t, srv, 0)

	// 断开后 Send/SendHex 不得 panic（修复前：send on closed channel）
	require.Error(t, tcpSess.Send("after-disconnect"))
	require.Error(t, tcpSess.SendHex("00ff"))
}
