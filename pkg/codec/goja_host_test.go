package codec_test

import (
	"errors"
	"fmt"
	"testing"

	"go-iot/pkg/codec"
	"go-iot/pkg/core"
	"go-iot/pkg/logger"
	"go-iot/pkg/store"
	_ "go-iot/pkg/timeseries"

	"github.com/dop251/goja"
	"github.com/stretchr/testify/require"
)

// 本文件锁定 goja 与 Go 宿主 API 的契约（版本/升级时勿静默破坏）：
// 1) 最后返回值为非 nil error → JS throw
// 2) nil error → 不中断后续语句
// 3) try/catch 可捕获
// 4) panic(vm.NewGoError) 与返回 error 语义一致
// 5) ScriptCodec.FuncInvoke 能收到异常并转为 Go error

func init() {
	logger.InitNop()
}

// hostAPI 模拟 DeviceOnline / SaveProperties 一类「(args) error」宿主方法
type hostAPI struct {
	calls int
}

func (h *hostAPI) Fail(msg string) error {
	h.calls++
	return fmt.Errorf("host fail: %s", msg)
}

func (h *hostAPI) Ok(msg string) error {
	h.calls++
	return nil
}

func (h *hostAPI) Echo(v string) (string, error) {
	h.calls++
	if v == "" {
		return "", errors.New("empty")
	}
	return "echo:" + v, nil
}

// panicHost 模拟 globe.throw：在 Go 内 panic(NewGoError)
type panicHost struct {
	vm *goja.Runtime
}

func (p *panicHost) FailWithPanic(msg string) {
	panic(p.vm.NewGoError(fmt.Errorf("panic host: %s", msg)))
}

func TestGoja_ErrorReturnThrowsInJS(t *testing.T) {
	vm := goja.New()
	h := &hostAPI{}
	require.NoError(t, vm.Set("host", h))

	_, err := vm.RunString(`host.Fail("x")`)
	require.Error(t, err, "non-nil error must become JS exception")
	require.Contains(t, err.Error(), "host fail: x")
	require.Equal(t, 1, h.calls)
}

func TestGoja_NilErrorDoesNotThrow(t *testing.T) {
	vm := goja.New()
	h := &hostAPI{}
	require.NoError(t, vm.Set("host", h))

	v, err := vm.RunString(`host.Ok("x"); "continued"`)
	require.NoError(t, err)
	require.Equal(t, "continued", v.String())
	require.Equal(t, 1, h.calls)
}

func TestGoja_ErrorReturnCanBeCaughtInJS(t *testing.T) {
	vm := goja.New()
	h := &hostAPI{}
	require.NoError(t, vm.Set("host", h))

	v, err := vm.RunString(`
		var caught = false;
		var msg = "";
		try {
			host.Fail("catch-me");
		} catch (e) {
			caught = true;
			msg = String(e);
		}
		({caught: caught, msg: msg, after: true})
	`)
	require.NoError(t, err)
	obj := v.ToObject(vm)
	require.Equal(t, true, obj.Get("caught").ToBoolean())
	require.Contains(t, obj.Get("msg").String(), "host fail: catch-me")
	require.Equal(t, true, obj.Get("after").ToBoolean())
}

func TestGoja_ErrorStopsSubsequentStatementsWithoutCatch(t *testing.T) {
	vm := goja.New()
	h := &hostAPI{}
	require.NoError(t, vm.Set("host", h))

	_, err := vm.RunString(`
		host.Fail("stop");
		host.Ok("should-not-run");
	`)
	require.Error(t, err)
	require.Equal(t, 1, h.calls, "second host call must not run after throw")
}

func TestGoja_ValueAndErrorReturn(t *testing.T) {
	vm := goja.New()
	h := &hostAPI{}
	require.NoError(t, vm.Set("host", h))

	// 成功：(string, nil) → JS 得到 string
	v, err := vm.RunString(`host.Echo("a")`)
	require.NoError(t, err)
	require.Equal(t, "echo:a", v.String())

	// 失败：(string, error) 且 error 非 nil → throw，不返回 string
	_, err = vm.RunString(`host.Echo("")`)
	require.Error(t, err)
	require.Contains(t, err.Error(), "empty")
}

func TestGoja_NewGoErrorPanicThrowsLikeErrorReturn(t *testing.T) {
	vm := goja.New()
	p := &panicHost{vm: vm}
	require.NoError(t, vm.Set("host", p))

	_, err := vm.RunString(`host.FailWithPanic("p")`)
	require.Error(t, err)
	require.Contains(t, err.Error(), "panic host: p")

	// try/catch 同样能接住
	v, err := vm.RunString(`
		var ok = false;
		try { host.FailWithPanic("c") } catch (e) { ok = true }
		ok
	`)
	require.NoError(t, err)
	require.Equal(t, true, v.ToBoolean())
}

func TestGoja_GlobeToCrc16StrThrowsOnBadInput(t *testing.T) {
	// 走真实 globe（pool 注册），非法 hex 应 throw
	pool, err := codec.NewVmPool(`function OnMessage(c){}`, 1)
	require.NoError(t, err)
	defer pool.Close()

	vm := pool.Get()
	defer pool.Put(vm)

	_, err = vm.RunString(`globe.ToCrc16Str("zz")`)
	require.Error(t, err, "invalid hex should throw via globe.throw / NewGoError")
}

func TestGoja_GlobeToCrc16StrSuccess(t *testing.T) {
	pool, err := codec.NewVmPool(`function OnMessage(c){}`, 1)
	require.NoError(t, err)
	defer pool.Close()

	vm := pool.Get()
	defer pool.Put(vm)

	v, err := vm.RunString(`globe.ToCrc16Str("0100de03e90302000500")`)
	require.NoError(t, err)
	require.Equal(t, "ce03", v.String())
}

func TestGoja_ScriptCodecFuncInvokeSurfacesJSException(t *testing.T) {
	script := `
function OnMessage(context) {
  throw "from-script"
}
`
	c, err := core.NewCodec("script_codec", "goja-contract-1", script)
	require.NoError(t, err)

	err = c.OnMessage(&core.BaseContext{DeviceId: "d1", ProductId: "goja-contract-1"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "from-script")
}

func TestGoja_ScriptCodecFuncInvokeSurfacesHostErrorReturn(t *testing.T) {
	// 通过 BaseContext.DeviceOnline 返回 error → goja throw → FuncInvoke err
	core.RegDeviceStore(store.NewMockDeviceStore())
	script := `
function OnMessage(context) {
  context.DeviceOnline("no-such-device")
}
`
	c, err := core.NewCodec("script_codec", "goja-contract-2", script)
	require.NoError(t, err)

	// 需要 Session 非 nil 才会走到「设备不存在」分支（否则 session is nil）
	// 这里不设 session：DeviceOnline 对存在设备会 session is nil；对不存在设备会 not exist
	sess := &fakeSess{info: map[string]any{}}
	err = c.OnMessage(&core.BaseContext{
		DeviceId:  "",
		ProductId: "goja-contract-2",
		Session:   sess,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "not exist or noActive")
}

func TestGoja_ScriptCodecMissingFunction(t *testing.T) {
	c, err := core.NewCodec("script_codec", "goja-contract-3", `function OnConnect(c){}`)
	require.NoError(t, err)
	err = c.OnMessage(&core.BaseContext{ProductId: "goja-contract-3"})
	require.ErrorIs(t, err, core.ErrFunctionNotImpl)
}

func TestGoja_ScriptCodecInvalidScriptAtCreate(t *testing.T) {
	_, err := core.NewCodec("script_codec", "goja-contract-4", `function OnMessage( {`)
	require.Error(t, err)
}

func TestGoja_TryCatchThenContinueAfterHostError(t *testing.T) {
	core.RegDeviceStore(store.NewMockDeviceStore())
	require.NoError(t, core.PutDevice(core.NewDevice("dev-ok", "p", 0)))

	script := `
var steps = [];
function OnMessage(context) {
  try {
    context.DeviceOnline("missing");
    steps.push("after-fail");
  } catch (e) {
    steps.push("caught");
  }
  context.DeviceOnline("dev-ok");
  steps.push("online-ok");
}
`
	c, err := core.NewCodec("script_codec", "goja-contract-5", script)
	require.NoError(t, err)

	sess := &fakeSess{info: map[string]any{}}
	err = c.OnMessage(&core.BaseContext{
		ProductId: "goja-contract-5",
		Session:   sess,
	})
	require.NoError(t, err, "caught host error should not fail OnMessage")
	require.Equal(t, "dev-ok", sess.deviceId)
	require.NotNil(t, core.GetSession("dev-ok"))
}

// fakeSess 最小 Session，供 DeviceOnline 使用
type fakeSess struct {
	deviceId string
	info     map[string]any
}

func (s *fakeSess) Disconnect() error          { return nil }
func (s *fakeSess) GetDeviceId() string         { return s.deviceId }
func (s *fakeSess) SetDeviceId(id string)       { s.deviceId = id }
func (s *fakeSess) Close() error                { return nil }
func (s *fakeSess) GetConInfo() map[string]any  { return s.info }
