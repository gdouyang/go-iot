package mqtt5

import (
	"testing"

	"go-iot/pkg/core"
	"go-iot/pkg/network/testhelper"
	"go-iot/pkg/store"

	logs "go-iot/pkg/logger"

	"github.com/stretchr/testify/require"
)

// authContext.DeviceOnline 单元：不启 broker（成功上线需真实 mqtt.Client，见集成测）。
func TestAuthContext_DeviceOnline_EmptyAndMissing(t *testing.T) {
	logs.InitNop()
	core.RegDeviceStore(store.NewMockDeviceStore())
	testhelper.SetupProductDevice(t, "m5-auth-p", "m5-auth-dev")

	ctx := &authContext{
		BaseContext: core.BaseContext{ProductId: "m5-auth-p"},
		rawClientId: "m5-auth-dev",
	}

	require.NoError(t, ctx.DeviceOnline(""))
	require.Equal(t, byte(0), ctx.authFailCode)

	err := ctx.DeviceOnline("missing")
	require.Error(t, err)
	require.Contains(t, err.Error(), "not exist or noActive")
	require.NotZero(t, ctx.authFailCode)
}
