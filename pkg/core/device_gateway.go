package core

import (
	"context"
	"time"

	"go-iot/pkg/common"
)

// InvokeOptions 功能调用路由选项。
type InvokeOptions struct {
	// OfflineCache 设备离线时写入离线命令队列。
	OfflineCache bool
	// ForceLocal 强制本机执行（集群内部 RPC 入口、无状态协议等）。
	ForceLocal bool
	// Timeout 跨节点 RPC 超时；0 表示默认 10s。
	Timeout time.Duration
}

// DeviceGateway 设备下行统一入口：本地会话 / 跨节点 RPC / 离线队列。
// 业务（API、规则）应优先走 Gateway，避免散落 cluster if。
type DeviceGateway interface {
	Invoke(ctx context.Context, msg FuncInvoke, opt InvokeOptions) *common.Err
}

// NodeInvoker 跨节点同步调用（请求-响应）。实现由 cluster 包提供（HTTP 等）。
type NodeInvoker interface {
	Enabled() bool
	LocalID() string
	// InvokeRemote 在目标节点本地执行功能调用并返回结果。
	InvokeRemote(ctx context.Context, nodeID string, msg FuncInvoke, timeout time.Duration) *common.Err
}

type defaultDeviceGateway struct{}

var (
	defaultGateway     DeviceGateway = defaultDeviceGateway{}
	defaultNodeInvoker NodeInvoker
)

// RegDeviceGateway 替换默认 Gateway（测试可注入）。
func RegDeviceGateway(g DeviceGateway) {
	if g == nil {
		return
	}
	defaultGateway = g
}

// GetDeviceGateway 返回当前 Gateway。
func GetDeviceGateway() DeviceGateway {
	return defaultGateway
}

// RegNodeInvoker 注册跨节点调用器（App.New 注入）。
func RegNodeInvoker(inv NodeInvoker) {
	defaultNodeInvoker = inv
}

// GetNodeInvoker 返回当前 NodeInvoker（可为 nil）。
func GetNodeInvoker() NodeInvoker {
	return defaultNodeInvoker
}

func (defaultDeviceGateway) Invoke(ctx context.Context, msg FuncInvoke, opt InvokeOptions) *common.Err {
	if ctx == nil {
		ctx = context.Background()
	}
	if opt.Timeout <= 0 {
		opt.Timeout = 10 * time.Second
	}

	device := GetDevice(msg.DeviceId)
	if device == nil {
		// 与本地路径一致：由 doCmdInvoke 返回「设备不存在」
		if opt.OfflineCache {
			_, err := DoCmdInvokeOffline(msg)
			return err
		}
		return DoCmdInvoke(msg)
	}

	invoker := defaultNodeInvoker
	remote := !opt.ForceLocal &&
		invoker != nil && invoker.Enabled() &&
		len(device.ClusterId) > 0 &&
		device.ClusterId != invoker.LocalID()

	if remote {
		// 离线缓存使用共享 Redis，本节点即可入队，无需 RPC。
		if opt.OfflineCache {
			state := GetDeviceState(msg.DeviceId, device.ProductId)
			if state == OFFLINE || IsDeviceDisconnect(msg.DeviceId) {
				_, err := DoCmdInvokeOffline(msg)
				return err
			}
		}
		// 同步跨节点：请求-响应；失败返回错误，不再回落本机乱执行。
		msg.ClusterId = device.ClusterId
		return invoker.InvokeRemote(ctx, device.ClusterId, msg, opt.Timeout)
	}

	if opt.OfflineCache {
		_, err := DoCmdInvokeOffline(msg)
		return err
	}
	return DoCmdInvoke(msg)
}

// DoCmdInvokeCluster 跨节点安全的功能调用（规则引擎等）。
// 已改为走 DeviceGateway（HTTP RPC），不再使用 Redis Pub/Sub。
func DoCmdInvokeCluster(message FuncInvoke) {
	_ = GetDeviceGateway().Invoke(context.Background(), message, InvokeOptions{
		OfflineCache: message.OfflineCache,
	})
}
