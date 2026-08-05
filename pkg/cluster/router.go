package cluster

import (
	"net/http"

	"go-iot/pkg/common"
	"go-iot/pkg/core"
)

// DeviceOwner 设备会话归属节点（基于 Redis/store 中的 ClusterId）。
type DeviceOwner struct {
	// Local 是否应由本节点处理。
	Local bool
	// NodeID 归属节点 name；Local 时为本节点。
	NodeID string
	// Device 运行时设备（可能为 nil）。
	Device *core.Device
}

// ResolveDeviceOwner 解析设备会话节点。
// 无集群 / 无设备 / 无 ClusterId / ClusterId==本机 → Local。
func ResolveDeviceOwner(deviceId string) DeviceOwner {
	if !Enabled() {
		return DeviceOwner{Local: true, NodeID: GetClusterId(), Device: core.GetDevice(deviceId)}
	}
	dev := core.GetDevice(deviceId)
	localID := GetClusterId()
	if dev == nil || len(dev.ClusterId) == 0 || dev.ClusterId == localID {
		return DeviceOwner{Local: true, NodeID: localID, Device: dev}
	}
	return DeviceOwner{Local: false, NodeID: dev.ClusterId, Device: dev}
}

// TryProxyDeviceRequest 若设备会话在其它节点，则把当前 HTTP 请求原样转发并返回响应。
// handled=false 表示应在本机继续处理。
func TryProxyDeviceRequest(deviceId string, req *http.Request) (handled bool, resp *common.JsonResp, err error) {
	owner := ResolveDeviceOwner(deviceId)
	if owner.Local {
		return false, nil, nil
	}
	resp, err = SingleInvoke(owner.NodeID, req)
	return true, resp, err
}

// ProxyOrLocal 便于 controller 写法：若已转发则 resp 非 nil。
// 返回 (resp, err)；resp==nil && err==nil 表示未转发。
func ProxyOrLocal(deviceId string, req *http.Request) (*common.JsonResp, error) {
	handled, resp, err := TryProxyDeviceRequest(deviceId, req)
	if !handled {
		return nil, nil
	}
	return resp, err
}
