package core_test

import (
	"testing"

	"go-iot/pkg/core"
	"go-iot/pkg/logger"
	"go-iot/pkg/store"

	"github.com/stretchr/testify/require"
)

func TestDeviceGetConfigFallsBackWhenDeviceValueEmpty(t *testing.T) {
	logger.InitNop()
	core.RegDeviceStore(store.NewMockDeviceStore())
	p, err := core.NewProduct("p-cfg", map[string]string{"username": "admin", "password": "123"}, core.TIME_SERISE_MOCK, `{"properties":[]}`)
	require.NoError(t, err)
	require.NoError(t, core.PutProduct(p))

	dev := core.NewDevice("D1", "p-cfg", 1)
	dev.Config = map[string]string{"username": "", "password": ""}
	require.Equal(t, "admin", dev.GetConfig("username"))
	require.Equal(t, "123", dev.GetConfig("password"))

	dev.Config["username"] = "devu"
	require.Equal(t, "devu", dev.GetConfig("username"))
	require.Equal(t, "123", dev.GetConfig("password"))
}
