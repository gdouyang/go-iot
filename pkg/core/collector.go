package core

import (
	"errors"
	"strings"
	"sync"

	"go-iot/pkg/tsl"
)

// CollectorRuntime 点表运行时。按网络类型注册，产品/设备服务只依赖此接口。
type CollectorRuntime interface {
	// Reload 从存储刷新本机点表缓存，并重载该产品下采集会话。
	Reload(productId string)
	// ReloadDevice 设备配置变更后重载该设备采集会话。
	ReloadDevice(deviceId string)
	// Invalidate 丢掉本机点表缓存。
	Invalidate(productId string)
	// CollectNow 立即采集一次；groupId 空则采集全部组。
	CollectNow(deviceId, groupId string) error
	// Normalize 解析点表 JSON、按物模型校验，返回应落库的规范 JSON。
	// 空配置返回 nil, nil（调用方删存储）。
	Normalize(raw []byte, tslData *tsl.TslData) ([]byte, error)
}

var (
	ErrCollectorNotActive    = errors.New("collector is not active")
	ErrCollectorNotSupported = errors.New("collector is not supported for this network type")
	collectorJSONLoader      func(productId string) (string, error)
	collectorJSONLoaderMu    sync.RWMutex
)

type noopCollectorRuntime struct{}

func (noopCollectorRuntime) Reload(string)       {}
func (noopCollectorRuntime) ReloadDevice(string) {}
func (noopCollectorRuntime) Invalidate(string)   {}
func (noopCollectorRuntime) CollectNow(string, string) error {
	return ErrCollectorNotActive
}

func (noopCollectorRuntime) Normalize(raw []byte, _ *tsl.TslData) ([]byte, error) {
	if collectorJSONBlank(raw) {
		return nil, nil
	}
	return nil, ErrCollectorNotSupported
}

func collectorJSONBlank(raw []byte) bool {
	s := strings.TrimSpace(string(raw))
	return s == "" || s == "null" || s == "{}"
}

// SetCollectorJSONLoader 注入点表存储读取（产品服务在 init 设置）。fn 为 nil 则清空。
func SetCollectorJSONLoader(fn func(productId string) (string, error)) {
	collectorJSONLoaderMu.Lock()
	collectorJSONLoader = fn
	collectorJSONLoaderMu.Unlock()
}

// LoadCollectorJSON 读产品点表原文。未注入 loader 或 productId 空时返回空串。
func LoadCollectorJSON(productId string) (string, error) {
	if productId == "" {
		return "", nil
	}
	collectorJSONLoaderMu.RLock()
	fn := collectorJSONLoader
	collectorJSONLoaderMu.RUnlock()
	if fn == nil {
		return "", nil
	}
	return fn(productId)
}

var (
	noopCollector     CollectorRuntime = noopCollectorRuntime{}
	collectorRuntimes sync.Map         // networkType -> CollectorRuntime
)

func normalizeCollectorNetType(networkType string) string {
	return strings.ToUpper(strings.TrimSpace(networkType))
}

// RegCollectorRuntime 按网络类型注册采集运行时。r 为 nil 则取消注册。
func RegCollectorRuntime(networkType string, r CollectorRuntime) {
	key := normalizeCollectorNetType(networkType)
	if key == "" {
		return
	}
	if r == nil {
		collectorRuntimes.Delete(key)
		return
	}
	collectorRuntimes.Store(key, r)
}

// GetCollectorRuntime 按网络类型取采集运行时。未注册返回空实现。
func GetCollectorRuntime(networkType string) CollectorRuntime {
	key := normalizeCollectorNetType(networkType)
	if key == "" {
		return noopCollector
	}
	if v, ok := collectorRuntimes.Load(key); ok {
		if rt, ok := v.(CollectorRuntime); ok && rt != nil {
			return rt
		}
	}
	return noopCollector
}
