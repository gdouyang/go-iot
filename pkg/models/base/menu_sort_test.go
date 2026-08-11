package base

import (
	"go-iot/pkg/models"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSortMenuResourcesByImageOrder(t *testing.T) {
	// 与侧栏目标顺序一致：产品 → 设备 → OTA → 规则 → 告警 → 通知 → 角色 → 用户 → 网络 → 系统配置
	list := []models.MenuResource{
		{Id: 1, Code: "sys-config", Sort: 100},
		{Id: 2, Code: "product-mgr", Sort: 10},
		{Id: 3, Code: "ota-mgr", Sort: 30},
		{Id: 4, Code: "device-mgr", Sort: 20},
		{Id: 5, Code: "role-mgr", Sort: 70},
		{Id: 6, Code: "notify-config", Sort: 60},
		{Id: 7, Code: "rule-mgr", Sort: 40},
		{Id: 8, Code: "alarm-mgr", Sort: 50},
		{Id: 9, Code: "user-mgr", Sort: 80},
		{Id: 10, Code: "network-config", Sort: 90},
	}
	sortMenuResources(list)
	want := []string{
		"product-mgr", "device-mgr", "ota-mgr", "rule-mgr", "alarm-mgr",
		"notify-config", "role-mgr", "user-mgr", "network-config", "sys-config",
	}
	got := make([]string, len(list))
	for i, m := range list {
		got[i] = m.Code
	}
	require.Equal(t, want, got)
}

func TestGetPermissionsIncludesSort(t *testing.T) {
	list := []models.MenuResource{
		{Code: "device-mgr", Name: "设备管理", Sort: 20, Action: `[{"id":"query","name":"查询"}]`},
		{Code: "product-mgr", Name: "产品管理", Sort: 10, Action: `[{"id":"query","name":"查询"}]`},
	}
	perms := getPermissions(0, list, false)
	require.Len(t, perms, 2)
	require.Equal(t, "product-mgr", perms[0].PermissionId)
	require.Equal(t, int32(10), perms[0].Sort)
	require.Equal(t, "device-mgr", perms[1].PermissionId)
	require.Equal(t, int32(20), perms[1].Sort)
}
