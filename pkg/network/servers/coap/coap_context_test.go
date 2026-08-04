package coapserver

import (
	"testing"

	"go-iot/pkg/core"
	"go-iot/pkg/network/testhelper"
	"go-iot/pkg/store"

	logs "go-iot/pkg/logger"

	"github.com/stretchr/testify/require"
)

func TestCoapDeviceOnlineErrors(t *testing.T) {
	logs.InitNop()
	core.RegDeviceStore(store.NewMockDeviceStore())
	testhelper.SetupProductDevice(t, "coap-prod", "coap-dev")

	ctx := &coapContext{
		BaseContext: core.BaseContext{ProductId: "coap-prod"},
	}
	require.NoError(t, ctx.DeviceOnline(""))
	require.Error(t, ctx.DeviceOnline("missing"))
	require.NoError(t, core.PutDevice(core.NewDevice("x", "y", 0)))
	require.Error(t, ctx.DeviceOnline("x"))
	require.NoError(t, ctx.DeviceOnline("coap-dev"))
}

func TestCoapDeviceOfflineErrors(t *testing.T) {
	logs.InitNop()
	core.RegDeviceStore(store.NewMockDeviceStore())
	testhelper.SetupProductDevice(t, "coap-prod2", "coap-dev2")
	ctx := &coapContext{BaseContext: core.BaseContext{ProductId: "coap-prod2"}}
	require.NoError(t, ctx.DeviceOffline(""))
	require.Error(t, ctx.DeviceOffline("missing"))
	require.NoError(t, ctx.DeviceOffline("coap-dev2"))
}
