package mqtt5

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/url"
	"testing"
	"time"

	_ "go-iot/pkg/codec"
	"go-iot/pkg/core"
	"go-iot/pkg/network/testhelper"
	"go-iot/pkg/option"
	_ "go-iot/pkg/timeseries"

	"github.com/eclipse/paho.golang/autopaho"
	"github.com/eclipse/paho.golang/paho"
	"github.com/stretchr/testify/require"
)

// startPlatformBroker 启动内置 broker（随机端口，clientID 即设备 ID）：
// 与 servers/mqtt5 不同，platform 版 OnConnectAuthenticate 要求 clientID 是已注册设备
func startPlatformBroker(t *testing.T, productId string, deviceIds ...string) int32 {
	t.Helper()
	// 注意不能循环调 testhelper.SetupProductDevice：每次都会 InitCore 重置
	// DeviceStore，前面注册的设备会被清空；InitCore 一次后批量 PutDevice
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
  context.DeviceOnline(context.GetClientId())
}
function OnMessage(context) {}
`
	_, err = core.NewCodec("script_codec", productId, script)
	require.NoError(t, err)
	// 不走 Parse()（会解析 go test 命令行参数）：直接构造默认配置
	option.Global = option.New()
	option.Global.Mqtt.Host = "127.0.0.1"
	option.Global.Mqtt.Port = int(testhelper.FreeTCPPort(t))
	require.NoError(t, Start())
	t.Cleanup(func() { _ = Stop() })
	return broker.spec.Port
}

// platformConnect 连接内置 broker（OnConnectError 不写日志，Stop 后重连失败安全）
func platformConnect(t *testing.T, port int32, clientID string) (*autopaho.ConnectionManager, context.CancelFunc) {
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

func waitTotalConnection(t *testing.T, want int32) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		if broker.TotalConnection() == want {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("TotalConnection=%d, want %d", broker.TotalConnection(), want)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// TestGoiotMqtt5_StopWithActiveConnections 覆盖内置 broker 的 Stop 死锁场景
// （servers/mqtt5 的 Stop/OnDisconnect 修复同源，此用例验证 platform 版）
func TestGoiotMqtt5_StopWithActiveConnections(t *testing.T) {
	productId := fmt.Sprintf("pf-m5-stop-%d", time.Now().UnixNano()%100000)
	deviceIds := []string{"pf-stop-0", "pf-stop-1"}
	port := startPlatformBroker(t, productId, deviceIds...)

	cancels := make([]context.CancelFunc, 0, len(deviceIds))
	for _, did := range deviceIds {
		_, cancel := platformConnect(t, port, did)
		cancels = append(cancels, cancel)
	}
	defer func() {
		for _, cancel := range cancels {
			cancel()
		}
	}()
	time.Sleep(300 * time.Millisecond)
	require.EqualValues(t, len(deviceIds), broker.TotalConnection())
	for _, did := range deviceIds {
		require.NotNil(t, core.GetSession(did))
	}

	done := make(chan struct{})
	go func() {
		_ = Stop()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(15 * time.Second):
		t.Fatal("platform broker Stop deadlocked with active connections")
	}
	require.EqualValues(t, 0, broker.TotalConnection())
	for _, did := range deviceIds {
		require.Nil(t, core.GetSession(did))
	}
}

// TestGoiotMqtt5_DisconnectCleanup 覆盖内置 broker 的设备主动断开清理
func TestGoiotMqtt5_DisconnectCleanup(t *testing.T) {
	productId := fmt.Sprintf("pf-m5-disc-%d", time.Now().UnixNano()%100000)
	port := startPlatformBroker(t, productId, "pf-disc-0")

	c, cancel := platformConnect(t, port, "pf-disc-0")
	defer cancel()
	time.Sleep(300 * time.Millisecond)
	require.EqualValues(t, 1, broker.TotalConnection())
	require.NotNil(t, core.GetSession("pf-disc-0"))

	require.NoError(t, c.Disconnect(context.Background()))
	waitTotalConnection(t, 0)
	require.Nil(t, core.GetSession("pf-disc-0"))
}

// TestGoiotMqtt5_DuplicateClientID 覆盖内置 broker 的同 clientID 重复连接路径
func TestGoiotMqtt5_DuplicateClientID(t *testing.T) {
	productId := fmt.Sprintf("pf-m5-dup-%d", time.Now().UnixNano()%100000)
	port := startPlatformBroker(t, productId, "pf-dup-0")

	_, cancel1 := platformConnect(t, port, "pf-dup-0")
	defer cancel1()
	time.Sleep(200 * time.Millisecond)
	require.EqualValues(t, 1, broker.TotalConnection())

	// 同 ID 第二个连接：attachClient 触发旧连接 OnDisconnect（原锁重入死锁点）
	_, cancel2 := platformConnect(t, port, "pf-dup-0")
	defer cancel2()

	// 互相踢 + autopaho 自动重连，连接数动态波动；断言能恢复在线且无堆积
	deadline := time.Now().Add(10 * time.Second)
	for {
		n := broker.TotalConnection()
		if n >= 1 {
			require.LessOrEqual(t, n, int32(2))
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("duplicate clientID connections not recovered, TotalConnection=%d", n)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// rawPlatformMqtt5Connect 手动构造 MQTT5 CONNECT 包并返回原始 TCP 连接：
// 用于模拟异常断开（不发 DISCONNECT 直接 Close）
func rawPlatformMqtt5Connect(t *testing.T, port int32, clientID string) net.Conn {
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

// TestGoiotMqtt5_AbruptDisconnect 覆盖异常断开（TCP 直接关闭、不发 DISCONNECT）：
// broker 通过读 EOF 触发 OnDisconnect(err)，必须完成 map 删除 + session 下线
func TestGoiotMqtt5_AbruptDisconnect(t *testing.T) {
	productId := fmt.Sprintf("pf-m5-abrupt-%d", time.Now().UnixNano()%100000)
	port := startPlatformBroker(t, productId, "pf-abrupt-0")

	conn := rawPlatformMqtt5Connect(t, port, "pf-abrupt-0")
	time.Sleep(300 * time.Millisecond)
	require.EqualValues(t, 1, broker.TotalConnection())
	require.NotNil(t, core.GetSession("pf-abrupt-0"))

	// 异常断开：直接关闭 TCP，不发 DISCONNECT
	require.NoError(t, conn.Close())
	deadline := time.Now().Add(5 * time.Second)
	for {
		if broker.TotalConnection() == 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("abrupt disconnect not cleaned up, TotalConnection=%d", broker.TotalConnection())
		}
		time.Sleep(50 * time.Millisecond)
	}
	require.Nil(t, core.GetSession("pf-abrupt-0"))
}
