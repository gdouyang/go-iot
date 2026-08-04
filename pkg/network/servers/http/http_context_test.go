package httpserver

import (
	"testing"

	"go-iot/pkg/core"
	"go-iot/pkg/network/testhelper"
	"go-iot/pkg/store"

	logs "go-iot/pkg/logger"

	"github.com/stretchr/testify/require"
)

func TestHttpDeviceOnlineErrors(t *testing.T) {
	logs.InitNop()
	core.RegDeviceStore(store.NewMockDeviceStore())
	testhelper.SetupProductDevice(t, "http-prod", "http-dev")

	ctx := &httpContext{
		BaseContext: core.BaseContext{ProductId: "http-prod"},
	}

	// 空 id：no-op
	require.NoError(t, ctx.DeviceOnline(""))

	// 设备不存在
	err := ctx.DeviceOnline("missing")
	require.Error(t, err)
	require.Contains(t, err.Error(), "is null")

	// 产品不一致
	require.NoError(t, core.PutDevice(core.NewDevice("other-dev", "other-prod", 0)))
	err = ctx.DeviceOnline("other-dev")
	require.Error(t, err)
	require.Contains(t, err.Error(), "product error")

	// 成功
	require.NoError(t, ctx.DeviceOnline("http-dev"))
}

func TestHttpDeviceOfflineErrors(t *testing.T) {
	logs.InitNop()
	core.RegDeviceStore(store.NewMockDeviceStore())
	testhelper.SetupProductDevice(t, "http-prod2", "http-dev2")

	ctx := &httpContext{
		BaseContext: core.BaseContext{ProductId: "http-prod2"},
	}
	require.NoError(t, ctx.DeviceOffline(""))
	require.Error(t, ctx.DeviceOffline("missing"))
	require.NoError(t, ctx.DeviceOffline("http-dev2"))
}
