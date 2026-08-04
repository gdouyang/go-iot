// Package testhelper 为 network 协议测试提供公共夹具（端口、产品/设备注册）。
package testhelper

import (
	"fmt"
	"net"
	"sync"
	"testing"

	"go-iot/pkg/core"
	"go-iot/pkg/store"

	logs "go-iot/pkg/logger"

	"github.com/stretchr/testify/require"
)

var setupOnce sync.Once

// InitCore 初始化 logger + mock DeviceStore（可重复调用）。
func InitCore() {
	setupOnce.Do(func() {
		logs.InitNop()
	})
	// 每次测试用新 store，避免设备串数据
	core.RegDeviceStore(store.NewMockDeviceStore())
}

// FreeTCPPort 分配本机空闲 TCP 端口。
func FreeTCPPort(t *testing.T) int32 {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer l.Close()
	return int32(l.Addr().(*net.TCPAddr).Port)
}

// FreeUDPPort 分配本机空闲 UDP 端口（CoAP）。
func FreeUDPPort(t *testing.T) int32 {
	t.Helper()
	c, err := net.ListenPacket("udp", "127.0.0.1:0")
	require.NoError(t, err)
	defer c.Close()
	return int32(c.LocalAddr().(*net.UDPAddr).Port)
}

// DefaultTSL 最小可用物模型。
func DefaultTSL() string {
	return `{"properties":[{"id":"temperature","type":"float"}],"events":[{"id":"alarm","type":"object","properties":[{"id":"level","type":"int"}]}],"functions":[{"id":"func1","inputs":[{"id":"name","type":"string"}]}]}`
}

// SetupProductDevice 注册产品与设备，返回 product。
func SetupProductDevice(t *testing.T, productId, deviceId string) *core.Product {
	t.Helper()
	InitCore()
	core.DeleteProduct(productId)
	core.DeleteDevice(deviceId)

	product, err := core.NewProduct(productId, map[string]string{}, core.TIME_SERISE_MOCK, DefaultTSL())
	require.NoError(t, err)
	require.NoError(t, core.PutProduct(product))
	require.NoError(t, core.PutDevice(core.NewDevice(deviceId, productId, 0)))
	return product
}

// OnlineScript 通用：上线 + 保存属性（HTTP/CoAP/TCP/WS/MQTT 脚本可复用）。
func OnlineScript(onlineExpr string) string {
	return fmt.Sprintf(`
function OnConnect(context) {
  console.log("OnConnect")
}
function OnMessage(context) {
  %s
  var data = JSON.parse(context.MsgToString())
  context.SaveProperties(data)
  if (context.GetSession && context.GetSession().Response) {
    try { context.GetSession().Response(JSON.stringify({ok:true})) } catch (e) {}
  }
  if (context.GetSession && context.GetSession().SendText) {
    try { context.GetSession().SendText(JSON.stringify({ok:true})) } catch (e) {}
  }
}
function OnInvoke(context) {
  console.log("OnInvoke")
}
`, onlineExpr)
}
