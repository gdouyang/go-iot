package modbus

import (
	"sync"

	"go-iot/pkg/core"
)

var collectorCache sync.Map // productId -> *CollectorConfig

func cloneCollector(c *CollectorConfig) *CollectorConfig {
	if c == nil {
		return &CollectorConfig{}
	}
	out := *c
	if c.Groups != nil {
		out.Groups = append([]Group(nil), c.Groups...)
	}
	if c.Points != nil {
		out.Points = append([]Point(nil), c.Points...)
	}
	return &out
}

func loadCollector(productId string) *CollectorConfig {
	raw, err := core.LoadCollectorJSON(productId)
	if err != nil {
		return &CollectorConfig{}
	}
	cfg, err := ParseCollectorJSON(raw)
	if err != nil || cfg == nil {
		return &CollectorConfig{}
	}
	return cfg
}

// GetCollector 本机缓存，未命中则读库。返回值勿修改。
func GetCollector(productId string) *CollectorConfig {
	if v, ok := collectorCache.Load(productId); ok {
		if cfg, ok := v.(*CollectorConfig); ok && cfg != nil {
			return cfg
		}
		return &CollectorConfig{}
	}
	cfg := loadCollector(productId)
	collectorCache.Store(productId, cfg)
	return cfg
}

func PutCollector(productId string, cfg *CollectorConfig) {
	if productId == "" {
		return
	}
	collectorCache.Store(productId, cloneCollector(cfg))
}

func DeleteCollector(productId string) {
	collectorCache.Delete(productId)
}

func reloadCollectorCache(productId string) *CollectorConfig {
	collectorCache.Delete(productId)
	return GetCollector(productId)
}
