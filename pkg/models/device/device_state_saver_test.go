package models

import (
	"testing"

	"go-iot/pkg/core"
	"go-iot/pkg/es/orm"
	"go-iot/pkg/logger"
	"go-iot/pkg/models"

	"github.com/stretchr/testify/require"
)

func TestUpdateOnlineStatusListSkipsWhenDeviceModelUnregistered(t *testing.T) {
	logger.InitNop()
	require.False(t, orm.IsRegistered(models.Device{}))
	err := UpdateOnlineStatusList([]string{"dev-1"}, core.ONLINE)
	require.ErrorIs(t, err, orm.ErrNotModel)
}

func TestUpdateOnlineStatusDoesNotPanicWhenUnregistered(t *testing.T) {
	logger.InitNop()
	require.NotPanics(t, func() {
		updateOnlineStatus([]deviceState{{
			deviceId:  "dev-unreg",
			productId: "prod-unreg",
			state:     core.ONLINE,
		}}, core.ONLINE)
	})
}

func TestUpdateOnlineStatusListRejectsEmptyArgs(t *testing.T) {
	require.Error(t, UpdateOnlineStatusList(nil, core.ONLINE))
	require.Error(t, UpdateOnlineStatusList([]string{"id"}, ""))
}
