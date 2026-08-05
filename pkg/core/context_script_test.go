package core_test

import (
	"testing"

	"go-iot/pkg/core"
	"go-iot/pkg/store"

	logs "go-iot/pkg/logger"

	// 注册 mock 时序实现
	_ "go-iot/pkg/timeseries"

	"github.com/dop251/goja"
	"github.com/stretchr/testify/require"
)

// fakeSession 仅用于 DeviceOnline 等单测
type fakeSession struct {
	deviceId string
	info     map[string]any
}

func newFakeSession() *fakeSession {
	return &fakeSession{info: map[string]any{}}
}

func (s *fakeSession) Disconnect() error           { return nil }
func (s *fakeSession) GetDeviceId() string         { return s.deviceId }
func (s *fakeSession) SetDeviceId(deviceId string) { s.deviceId = deviceId }
func (s *fakeSession) Close() error                { return nil }
func (s *fakeSession) GetConInfo() map[string]any  { return s.info }

func setupProduct(t *testing.T, productId string) *core.Product {
	t.Helper()
	logs.InitNop()
	core.RegDeviceStore(store.NewMockDeviceStore())
	product, err := core.NewProduct(productId, map[string]string{}, core.TIME_SERISE_MOCK,
		`{"properties":[{"id":"temperature","type":"float"}],"events":[{"id":"alarm","type":"object","properties":[{"id":"level","type":"int"}]}],"functions":[]}`)
	require.NoError(t, err)
	require.NoError(t, core.PutProduct(product))
	return product
}

// ---- SaveProperties ----

func TestSavePropertiesEmptyDeviceIdReturnsError(t *testing.T) {
	setupProduct(t, "p-sp-empty")
	ctx := &core.BaseContext{ProductId: "p-sp-empty", DeviceId: ""}
	err := ctx.SaveProperties(map[string]any{"temperature": 1.2})
	require.Error(t, err)
	require.Contains(t, err.Error(), "deviceId is empty")
}

func TestSavePropertiesProductMissingReturnsError(t *testing.T) {
	logs.InitNop()
	core.RegDeviceStore(store.NewMockDeviceStore())
	ctx := &core.BaseContext{ProductId: "no-such-product", DeviceId: "dev-1"}
	err := ctx.SaveProperties(map[string]any{"temperature": 1.2})
	require.Error(t, err)
	require.Contains(t, err.Error(), "not exist or noActive")
}

func TestSavePropertiesErrorThrowsInJS(t *testing.T) {
	setupProduct(t, "p-sp-js-err")
	ctx := &core.BaseContext{ProductId: "p-sp-js-err", DeviceId: ""}
	vm := goja.New()
	require.NoError(t, vm.Set("ctx", ctx))
	_, err := vm.RunString(`ctx.SaveProperties({temperature: 1})`)
	require.Error(t, err, "goja should throw when SaveProperties returns error")
}

func TestSavePropertiesSuccessInJS(t *testing.T) {
	setupProduct(t, "p-sp-js-ok")
	ctx := &core.BaseContext{ProductId: "p-sp-js-ok", DeviceId: "dev-1"}
	vm := goja.New()
	require.NoError(t, vm.Set("ctx", ctx))
	_, err := vm.RunString(`ctx.SaveProperties({temperature: 1}); var ok = true; ok`)
	require.NoError(t, err)
}

// ---- SaveEvents ----

func TestSaveEventsEmptyDeviceIdReturnsError(t *testing.T) {
	setupProduct(t, "p-se-empty")
	ctx := &core.BaseContext{ProductId: "p-se-empty", DeviceId: ""}
	err := ctx.SaveEvents("alarm", map[string]any{"level": 1})
	require.Error(t, err)
	require.Contains(t, err.Error(), "deviceId is empty")
}

func TestSaveEventsProductMissingReturnsError(t *testing.T) {
	logs.InitNop()
	core.RegDeviceStore(store.NewMockDeviceStore())
	ctx := &core.BaseContext{ProductId: "no-product-evt", DeviceId: "dev-1"}
	err := ctx.SaveEvents("alarm", map[string]any{"level": 1})
	require.Error(t, err)
	require.Contains(t, err.Error(), "not exist or noActive")
}

func TestSaveEventsErrorThrowsInJS(t *testing.T) {
	setupProduct(t, "p-se-js-err")
	ctx := &core.BaseContext{ProductId: "p-se-js-err", DeviceId: ""}
	vm := goja.New()
	require.NoError(t, vm.Set("ctx", ctx))
	_, err := vm.RunString(`ctx.SaveEvents("alarm", {level: 1})`)
	require.Error(t, err, "goja should throw when SaveEvents returns error")
	t.Logf("goja err type: %T", err)
	t.Logf("goja err: %v", err)
	// 对照：纯 Go 调用返回的 error 文案（无 GoError 前缀）
	errGo := ctx.SaveEvents("alarm", map[string]any{"level": 1})
	require.Error(t, errGo)
	t.Logf("direct Go err: %v", errGo)
	require.Contains(t, err.Error(), "deviceId is empty")
	require.Contains(t, errGo.Error(), "deviceId is empty")
}

func TestSaveEventsSuccessInJS(t *testing.T) {
	setupProduct(t, "p-se-js-ok")
	ctx := &core.BaseContext{ProductId: "p-se-js-ok", DeviceId: "dev-1"}
	vm := goja.New()
	require.NoError(t, vm.Set("ctx", ctx))
	_, err := vm.RunString(`ctx.SaveEvents("alarm", {level: 2})`)
	require.NoError(t, err)
}

// ---- DeviceOnline ----

func TestDeviceOnlineEmptyIdNoError(t *testing.T) {
	logs.InitNop()
	core.RegDeviceStore(store.NewMockDeviceStore())
	ctx := &core.BaseContext{ProductId: "p", Session: newFakeSession()}
	require.NoError(t, ctx.DeviceOnline(""))
	require.NoError(t, ctx.DeviceOnline("   "))
}

func TestDeviceOnlineDeviceMissingReturnsError(t *testing.T) {
	logs.InitNop()
	core.RegDeviceStore(store.NewMockDeviceStore())
	sess := newFakeSession()
	ctx := &core.BaseContext{ProductId: "p", Session: sess}
	err := ctx.DeviceOnline("missing-device")
	require.Error(t, err)
	require.Contains(t, err.Error(), "not exist or noActive")
}

func TestDeviceOnlineSessionNilReturnsError(t *testing.T) {
	logs.InitNop()
	core.RegDeviceStore(store.NewMockDeviceStore())
	require.NoError(t, core.PutDevice(core.NewDevice("dev-online-1", "p", 0)))
	ctx := &core.BaseContext{ProductId: "p", Session: nil}
	err := ctx.DeviceOnline("dev-online-1")
	require.Error(t, err)
	require.Contains(t, err.Error(), "session is nil")
}

func TestDeviceOnlineSuccess(t *testing.T) {
	logs.InitNop()
	core.RegDeviceStore(store.NewMockDeviceStore())
	require.NoError(t, core.PutDevice(core.NewDevice("dev-online-ok", "p", 0)))
	sess := newFakeSession()
	ctx := &core.BaseContext{ProductId: "p", Session: sess}
	require.NoError(t, ctx.DeviceOnline("dev-online-ok"))
	require.Equal(t, "dev-online-ok", ctx.DeviceId)
	require.Equal(t, "dev-online-ok", sess.GetDeviceId())
	require.NotNil(t, core.GetSession("dev-online-ok"))
}

func TestDeviceOnlineErrorThrowsInJS(t *testing.T) {
	logs.InitNop()
	core.RegDeviceStore(store.NewMockDeviceStore())
	ctx := &core.BaseContext{ProductId: "p", Session: newFakeSession()}
	vm := goja.New()
	require.NoError(t, vm.Set("ctx", ctx))
	_, err := vm.RunString(`ctx.DeviceOnline("no-device")`)
	require.Error(t, err, "goja should throw when DeviceOnline returns error")
}

func TestDeviceOnlineSuccessInJS(t *testing.T) {
	logs.InitNop()
	core.RegDeviceStore(store.NewMockDeviceStore())
	require.NoError(t, core.PutDevice(core.NewDevice("dev-js-ok", "p", 0)))
	ctx := &core.BaseContext{ProductId: "p", Session: newFakeSession()}
	vm := goja.New()
	require.NoError(t, vm.Set("ctx", ctx))
	_, err := vm.RunString(`ctx.DeviceOnline("dev-js-ok")`)
	require.NoError(t, err)
}
