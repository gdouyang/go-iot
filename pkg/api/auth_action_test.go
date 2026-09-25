package api

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCreateActionIdIsAdd(t *testing.T) {
	require.Equal(t, "add", CreateAction.Id)
	require.Equal(t, "新增", CreateAction.Name)
}

func TestIsLicenseExemptPath(t *testing.T) {
	// 白名单路径应放行
	require.True(t, isLicenseExemptPath("/api/login"))
	require.True(t, isLicenseExemptPath("/login"))
	require.True(t, isLicenseExemptPath("/api/logout"))
	require.True(t, isLicenseExemptPath("/api/user-info"))
	require.True(t, isLicenseExemptPath("/api/system/license/status"))
	require.True(t, isLicenseExemptPath("/api/system/license/info"))
	require.True(t, isLicenseExemptPath("/api/system/license/upload"))

	// 业务接口应拦截
	require.False(t, isLicenseExemptPath("/api/device/page"))
	require.False(t, isLicenseExemptPath("/api/product/page"))
	require.False(t, isLicenseExemptPath("/api/network/page"))
	require.False(t, isLicenseExemptPath("/api/rule/page"))
	require.False(t, isLicenseExemptPath("/api/alarm/page"))
}
