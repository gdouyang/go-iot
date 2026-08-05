package core_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"go-iot/pkg/common"
	"go-iot/pkg/core"

	"github.com/stretchr/testify/require"
)

type recordingInvoker struct {
	localID   string
	enabled   bool
	lastNode  string
	lastMsg   core.FuncInvoke
	remoteErr *common.Err
	calls     int32
}

func (r *recordingInvoker) Enabled() bool   { return r.enabled }
func (r *recordingInvoker) LocalID() string { return r.localID }
func (r *recordingInvoker) InvokeRemote(ctx context.Context, nodeID string, msg core.FuncInvoke, timeout time.Duration) *common.Err {
	atomic.AddInt32(&r.calls, 1)
	r.lastNode = nodeID
	r.lastMsg = msg
	return r.remoteErr
}

func TestDeviceGateway_RemoteInvoke(t *testing.T) {
	inv := &recordingInvoker{localID: "node-a", enabled: true}
	core.RegNodeInvoker(inv)
	t.Cleanup(func() { core.RegNodeInvoker(nil) })

	mem := newMapDeviceStore()
	core.RegDeviceStore(mem)
	t.Cleanup(func() { core.RegDeviceStore(nil) })

	dev := core.NewDevice("dev-1", "prod", 0)
	dev.ClusterId = "node-b"
	require.NoError(t, core.PutDevice(dev))

	err := core.GetDeviceGateway().Invoke(context.Background(), core.FuncInvoke{
		DeviceId:   "dev-1",
		FunctionId: "f1",
	}, core.InvokeOptions{})
	require.Nil(t, err)
	require.Equal(t, int32(1), atomic.LoadInt32(&inv.calls))
	require.Equal(t, "node-b", inv.lastNode)
	require.Equal(t, "dev-1", inv.lastMsg.DeviceId)
}

func TestDeviceGateway_ForceLocalSkipsRemote(t *testing.T) {
	inv := &recordingInvoker{localID: "node-a", enabled: true, remoteErr: common.NewErr500("should not call")}
	core.RegNodeInvoker(inv)
	t.Cleanup(func() { core.RegNodeInvoker(nil) })

	mem := newMapDeviceStore()
	core.RegDeviceStore(mem)
	t.Cleanup(func() { core.RegDeviceStore(nil) })

	dev := core.NewDevice("dev-2", "prod", 0)
	dev.ClusterId = "node-b"
	require.NoError(t, core.PutDevice(dev))

	// ForceLocal: 走本机 DoCmdInvoke，设备无产品会返回业务错误，但不应 RPC
	err := core.GetDeviceGateway().Invoke(context.Background(), core.FuncInvoke{
		DeviceId:   "dev-2",
		FunctionId: "f1",
	}, core.InvokeOptions{ForceLocal: true})
	require.NotNil(t, err)
	require.Equal(t, int32(0), atomic.LoadInt32(&inv.calls))
}

func TestDeviceGateway_RemoteFailureNoLocalFallback(t *testing.T) {
	inv := &recordingInvoker{
		localID:   "node-a",
		enabled:   true,
		remoteErr: common.NewErr500("peer down"),
	}
	core.RegNodeInvoker(inv)
	t.Cleanup(func() { core.RegNodeInvoker(nil) })

	mem := newMapDeviceStore()
	core.RegDeviceStore(mem)
	t.Cleanup(func() { core.RegDeviceStore(nil) })

	dev := core.NewDevice("dev-3", "prod", 0)
	dev.ClusterId = "node-b"
	require.NoError(t, core.PutDevice(dev))

	err := core.GetDeviceGateway().Invoke(context.Background(), core.FuncInvoke{
		DeviceId:   "dev-3",
		FunctionId: "f1",
	}, core.InvokeOptions{})
	require.NotNil(t, err)
	require.Equal(t, 500, err.Code)
	require.Contains(t, err.Message, "peer down")
}

func TestDoCmdInvokeCluster_UsesGateway(t *testing.T) {
	inv := &recordingInvoker{localID: "node-a", enabled: true}
	core.RegNodeInvoker(inv)
	t.Cleanup(func() { core.RegNodeInvoker(nil) })

	mem := newMapDeviceStore()
	core.RegDeviceStore(mem)
	t.Cleanup(func() { core.RegDeviceStore(nil) })

	dev := core.NewDevice("dev-4", "prod", 0)
	dev.ClusterId = "node-b"
	require.NoError(t, core.PutDevice(dev))

	core.DoCmdInvokeCluster(core.FuncInvoke{DeviceId: "dev-4", FunctionId: "x"})
	require.Equal(t, int32(1), atomic.LoadInt32(&inv.calls))
}

// minimal DeviceStore for gateway tests
type mapDeviceStore struct {
	devices  map[string]*core.Device
	products map[string]*core.Product
}

func newMapDeviceStore() *mapDeviceStore {
	return &mapDeviceStore{
		devices:  map[string]*core.Device{},
		products: map[string]*core.Product{},
	}
}

func (m *mapDeviceStore) Id() string                       { return "map" }
func (m *mapDeviceStore) GetDevice(id string) *core.Device { return m.devices[id] }
func (m *mapDeviceStore) PutDevice(d *core.Device) error {
	m.devices[d.GetId()] = d
	return nil
}
func (m *mapDeviceStore) DelDevice(id string)                   { delete(m.devices, id) }
func (m *mapDeviceStore) RefreshOfflineTimeout(deviceId string) {}
func (m *mapDeviceStore) GetDeviceData(deviceId, key string) string {
	return ""
}
func (m *mapDeviceStore) SetDeviceData(deviceId, key string, val any) {}
func (m *mapDeviceStore) GetProduct(id string) *core.Product          { return m.products[id] }
func (m *mapDeviceStore) PutProduct(p *core.Product) error {
	m.products[p.GetId()] = p
	return nil
}
func (m *mapDeviceStore) DelProduct(id string) { delete(m.products, id) }
