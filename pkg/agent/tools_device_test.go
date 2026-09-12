package agent

import (
	"encoding/json"
	"errors"
	"testing"

	"go-iot/pkg/models"

	"github.com/stretchr/testify/require"
)

func TestCreateDeviceConfirmAndNormalize(t *testing.T) {
	origP := lookupProduct
	origD := lookupDevice
	origAdd := addDevice
	t.Cleanup(func() {
		lookupProduct = origP
		lookupDevice = origD
		addDevice = origAdd
	})
	lookupProduct = func(id string) (*models.ProductModel, error) {
		return &models.ProductModel{Product: models.Product{Id: id, CreateId: 1, NetworkType: "MQTT_BROKER"}}, nil
	}
	lookupDevice = func(id string) (*models.DeviceModel, error) {
		return nil, errors.New("not found")
	}
	var saved *models.DeviceModel
	addDevice = func(ob *models.DeviceModel) error {
		saved = ob
		return nil
	}

	g := BeforeToolCall("confirm", writeCtx(1), ToolCreateDevice, json.RawMessage(`{"id":"1234","name":"1234设备","productId":"TEST-MQTT"}`))
	require.Equal(t, ToolConfirm, g.Action)
	require.True(t, g.Terminate)

	out, pid, err := createDeviceTool{}.Execute(writeCtx(1), json.RawMessage(`{"id":"1234","name":"1234设备","productId":"TEST-MQTT"}`))
	require.NoError(t, err)
	require.Equal(t, "TEST-MQTT", pid)
	m := out.(map[string]any)
	require.Equal(t, "1234", m["deviceId"])
	require.NotNil(t, saved)
	require.Equal(t, "1234设备", saved.Name)
}

func TestCreateDeviceRejectsMissingProduct(t *testing.T) {
	stubNoProduct(t)
	g := BeforeToolCall("confirm", writeCtx(1), ToolCreateDevice, json.RawMessage(`{"id":"D1","name":"n","productId":"NOPE"}`))
	require.Equal(t, ToolBlock, g.Action)
}

func TestGetDevice(t *testing.T) {
	origP := lookupProduct
	origD := lookupDevice
	t.Cleanup(func() {
		lookupProduct = origP
		lookupDevice = origD
	})
	lookupProduct = func(id string) (*models.ProductModel, error) {
		return &models.ProductModel{Product: models.Product{Id: id, CreateId: 1, NetworkType: "MQTT_BROKER"}}, nil
	}
	lookupDevice = func(id string) (*models.DeviceModel, error) {
		if id == "1234" {
			return &models.DeviceModel{Device: models.Device{Id: "1234", Name: "温湿度01", ProductId: "TEST-MQTT", State: "online"}}, nil
		}
		return nil, errors.New("not found")
	}

	// 1. 只读门闩，confirm 模式下也应该是 allow
	g := BeforeToolCall("confirm", writeCtx(1), ToolGetDevice, json.RawMessage(`{"deviceId":"1234"}`))
	require.Equal(t, ToolAllow, g.Action)

	// 2. 执行查询存在设备
	out, pid, err := getDeviceTool{}.Execute(writeCtx(1), json.RawMessage(`{"deviceId":"1234"}`))
	require.NoError(t, err)
	require.Equal(t, "TEST-MQTT", pid)
	m := out.(map[string]any)
	require.True(t, m["ok"].(bool))
	require.Equal(t, "1234", m["deviceId"])
	require.Equal(t, "TEST-MQTT", m["productId"])
	require.NotEmpty(t, m["state"])

	// 3. 执行查询不存在设备
	out2, _, err := getDeviceTool{}.Execute(writeCtx(1), json.RawMessage(`{"deviceId":"not-exist"}`))
	require.NoError(t, err)
	m2 := out2.(map[string]any)
	require.False(t, m2["ok"].(bool))
	require.Contains(t, m2["message"], "设备不存在")
}

func TestListDevices(t *testing.T) {
	origP := lookupProduct
	origList := listDevices
	t.Cleanup(func() {
		lookupProduct = origP
		listDevices = origList
	})
	lookupProduct = func(id string) (*models.ProductModel, error) {
		return &models.ProductModel{Product: models.Product{Id: id, CreateId: 1, NetworkType: "MQTT_BROKER"}}, nil
	}
	listDevices = func(productId string, pageNum, pageSize int) (*models.PageResult[models.Device], error) {
		return &models.PageResult[models.Device]{
			TotalCount: 1,
			PageNum:    1,
			PageSize:   20,
			List: []models.Device{
				{Id: "1234", Name: "设备1", ProductId: productId, State: "online"},
			},
		}, nil
	}

	// 只读门闩
	g := BeforeToolCall("confirm", writeCtx(1), ToolListDevices, json.RawMessage(`{"productId":"TEST-MQTT"}`))
	require.Equal(t, ToolAllow, g.Action)

	out, pid, err := listDevicesTool{}.Execute(writeCtx(1), json.RawMessage(`{"productId":"TEST-MQTT"}`))
	require.NoError(t, err)
	require.Equal(t, "TEST-MQTT", pid)
	m := out.(map[string]any)
	require.True(t, m["ok"].(bool))
	require.Equal(t, int64(1), m["total"])
	list := m["list"].([]map[string]any)
	require.Len(t, list, 1)
	require.Equal(t, "1234", list[0]["deviceId"])
}
