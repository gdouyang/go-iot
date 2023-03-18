package models

import (
	"context"
	"encoding/json"
	"fmt"
	"go-iot/pkg/core"
	"go-iot/pkg/core/redis"
	"go-iot/pkg/core/util"
	"sync"
	"time"

	"github.com/beego/beego/v2/core/logs"
)

func init() {
	core.RegDeviceManager(&redisDeviceManager{cache: sync.Map{}})
	core.RegProductManager(&redisProductManager{cache: sync.Map{}})
}

// device manager for redis
type redisDeviceManager struct {
	cache sync.Map
}

func (p *redisDeviceManager) getKey(deviceId string) string {
	return "goiot:device:" + deviceId
}

func (m *redisDeviceManager) get(deviceId string) (*core.Device, bool) {
	device, ok := m.cache.Load(deviceId)
	if ok {
		if device != nil {
			return device.(*core.Device), true
		}
		return nil, true
	}
	return nil, false
}

func (p *redisDeviceManager) Id() string {
	return "redis"
}

func (m *redisDeviceManager) Get(deviceId string) *core.Device {
	device, ok := m.get(deviceId)
	if ok {
		return device
	}
	if device == nil {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
		defer cancel()
		rdb := redis.GetRedisClient()
		data, err := rdb.HGetAll(ctx, m.getKey(deviceId)).Result()
		if err != nil {
			logs.Error(err)
		}
		if len(data) == 0 {
			m.cache.Store(deviceId, nil)
			return nil
		}
		config := map[string]string{}
		if str, ok := data["config"]; ok {
			err = json.Unmarshal([]byte(str), &config)
			if err != nil {
				logs.Error("device config parse error:", err)
			}
		}
		dat := map[string]string{}
		if str, ok := data["data"]; ok {
			err = json.Unmarshal([]byte(str), &dat)
			if err != nil {
				logs.Error("device data parse error:", err)
			}
		}
		var createId int64
		if str, ok := data["createId"]; ok {
			createId, err = util.StringToInt64(str)
			if err != nil {
				logs.Error("device createId parse error:", err)
			}
		}
		dev := &core.Device{
			Id:        data["id"],
			ProductId: data["productId"],
			CreateId:  createId,
			Config:    config,
			Data:      dat,
		}
		m.cache.Store(deviceId, dev)
		return dev
	}
	return nil
}

func (m *redisDeviceManager) Put(device *core.Device) {
	p := device
	byt, _ := json.Marshal(p.Config)
	dat, _ := json.Marshal(p.Data)
	data := map[string]string{
		"id":        p.Id,
		"productId": p.ProductId,
		"devType":   p.DeviceType,
		"parentId":  p.ParentId,
		"createId":  fmt.Sprintf("%v", p.CreateId),
		"config":    string(byt),
		"data":      string(dat),
	}
	rdb := redis.GetRedisClient()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	err := rdb.HSet(ctx, m.getKey(p.Id), data).Err()
	if err != nil {
		logs.Error(err)
	}
	m.cache.Store(device.GetId(), device)
}

func (m *redisDeviceManager) Del(deviceId string) {
	rdb := redis.GetRedisClient()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	rdb.Del(ctx, m.getKey(deviceId))
	m.cache.Delete(deviceId)
}

// product manager for redis
type redisProductManager struct {
	cache sync.Map
}

func (p *redisProductManager) getKey(deviceId string) string {
	return "goiot:product:" + deviceId
}

func (m *redisProductManager) get(deviceId string) (*core.Product, bool) {
	product, ok := m.cache.Load(deviceId)
	if ok {
		if product != nil {
			return product.(*core.Product), true
		}
		return nil, true
	}
	return nil, false
}

func (p *redisProductManager) Id() string {
	return "redis"
}

func (m *redisProductManager) Get(productId string) *core.Product {
	product, ok := m.get(productId)
	if ok {
		return product
	}
	if product == nil {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
		defer cancel()
		rdb := redis.GetRedisClient()
		data, err := rdb.HGetAll(ctx, m.getKey(productId)).Result()
		if err != nil {
			logs.Error(err)
		}
		if len(data) == 0 {
			m.cache.Store(productId, nil)
			return nil
		}
		config := map[string]string{}
		if str, ok := data["config"]; ok {
			err = json.Unmarshal([]byte(str), &config)
			if err != nil {
				logs.Error("device config parse error:", err)
			}
		}
		produ, err := core.NewProduct(data["id"], config, data["storePolicy"], data["tslData"])
		if err != nil {
			logs.Error(err)
		} else {
			m.cache.Store(productId, produ)
			return produ
		}
	}
	return nil
}

func (m *redisProductManager) Put(product *core.Product) {
	if product == nil {
		panic("product not be nil")
	}
	if len(product.GetId()) == 0 {
		panic("product id must be present")
	}
	p := product
	byt, _ := json.Marshal(p.Config)
	data := map[string]string{
		"id":          p.Id,
		"storePolicy": p.StorePolicy,
		"config":      string(byt),
		"tslData":     p.TslData.Text,
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	rdb := redis.GetRedisClient()
	err := rdb.HSet(ctx, m.getKey(p.Id), data).Err()
	if err != nil {
		logs.Error(err)
	}
	m.cache.Store(product.GetId(), product)
}

func (m *redisProductManager) Del(productId string) {
	rdb := redis.GetRedisClient()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	rdb.Del(ctx, m.getKey(productId))
	m.cache.Delete(productId)
}
