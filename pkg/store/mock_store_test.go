package store

import (
	"testing"

	"go-iot/pkg/core"

	"github.com/stretchr/testify/require"
)

func TestMockDeviceStore_Id(t *testing.T) {
	s := NewMockDeviceStore()
	require.Equal(t, "mock", s.Id())
}

func TestMockDeviceStore_DeviceCRUD(t *testing.T) {
	s := NewMockDeviceStore()
	require.Nil(t, s.GetDevice("d1"))

	dev := core.NewDevice("d1", "p1", 9)
	dev.Name = "sensor"
	require.NoError(t, s.PutDevice(dev))
	got := s.GetDevice("d1")
	require.NotNil(t, got)
	require.Equal(t, "d1", got.Id)
	require.Equal(t, "p1", got.ProductId)
	require.Equal(t, "sensor", got.Name)
	require.Equal(t, int64(9), got.CreateId)

	s.DelDevice("d1")
	require.Nil(t, s.GetDevice("d1"))
}

func TestMockDeviceStore_PutDeviceValidation(t *testing.T) {
	s := NewMockDeviceStore()
	require.Error(t, s.PutDevice(nil))
	require.Error(t, s.PutDevice(core.NewDevice("", "p", 0)))
}

func TestMockDeviceStore_DeviceData(t *testing.T) {
	s := NewMockDeviceStore()
	require.Equal(t, "", s.GetDeviceData("d1", "k"))
	s.SetDeviceData("d1", "k", "v1")
	require.Equal(t, "v1", s.GetDeviceData("d1", "k"))
	s.SetDeviceData("d1", "n", 42)
	require.Equal(t, "42", s.GetDeviceData("d1", "n"))
	// 未知 key
	require.Equal(t, "<nil>", s.GetDeviceData("d1", "missing"))
}

func TestMockDeviceStore_ProductCRUD(t *testing.T) {
	s := NewMockDeviceStore()
	require.Nil(t, s.GetProduct("p1"))

	p, err := core.NewProduct("p1", map[string]string{"a": "1"}, "es", "")
	require.NoError(t, err)
	require.NoError(t, s.PutProduct(p))
	got := s.GetProduct("p1")
	require.NotNil(t, got)
	require.Equal(t, "p1", got.Id)
	require.Equal(t, "es", got.StorePolicy)

	s.DelProduct("p1")
	require.Nil(t, s.GetProduct("p1"))
}

func TestMockDeviceStore_PutProductValidation(t *testing.T) {
	s := NewMockDeviceStore()
	require.Error(t, s.PutProduct(nil))
	p, err := core.NewProduct("", nil, "", "")
	require.NoError(t, err)
	require.Error(t, s.PutProduct(p))
}

func TestMockDeviceStore_RefreshOfflineTimeoutNoop(t *testing.T) {
	// mock 实现为空操作，确保不 panic
	NewMockDeviceStore().RefreshOfflineTimeout("any")
}
