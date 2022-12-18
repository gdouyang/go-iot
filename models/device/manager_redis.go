package models

import (
	"context"
	"encoding/json"
	"go-iot/codec"
	"sync"
	"time"

	"github.com/beego/beego/v2/core/logs"
)

func init() {
	codec.RegDeviceManager(&redisDeviceManager{cache: make(map[string]codec.Device)})
	codec.RegProductManager(&redisProductManager{cache: make(map[string]codec.Product)})
}

// redisDeviceManager
type redisDeviceManager struct {
	sync.RWMutex
	cache map[string]codec.Device
}

func (p *redisDeviceManager) Id() string {
	return "redis"
}

func (m *redisDeviceManager) Get(deviceId string) codec.Device {
	device, ok := m.cache[deviceId]
	if ok {
		return device
	}
	if device == nil {
		m.Lock()
		defer m.Unlock()
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
		defer cancel()
		rdb := codec.GetRedisClient()
		data, err := rdb.HGetAll(ctx, "goiot:product:"+deviceId).Result()
		if err != nil {
			logs.Error(err)
		}
		if data == nil {
			m.cache[deviceId] = nil
			return nil
		}
		config := map[string]string{}
		err = json.Unmarshal([]byte(data["config"]), &config)
		if err != nil {
			logs.Error(err)
		}
		dat := map[string]string{}
		err = json.Unmarshal([]byte(data["data"]), &dat)
		if err != nil {
			logs.Error(err)
		}
		dev := &codec.DefaultDevice{
			Id:        data["id"],
			ProductId: data["product"],
			Config:    config,
			Data:      dat,
		}
		m.cache[deviceId] = dev
		return dev
	}
	return nil
}

func (m *redisDeviceManager) Put(device codec.Device) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	p := device.(*codec.DefaultDevice)
	byt, _ := json.Marshal(p.Config)
	dat, _ := json.Marshal(p.Data)
	data := map[string]string{
		"id":        p.Id,
		"config":    string(byt),
		"productId": p.ProductId,
		"data":      string(dat),
	}
	rdb := codec.GetRedisClient()
	rdb.HSet(ctx, "goiot:device:"+p.Id, data).Err()
	m.cache[device.GetId()] = device
}

type redisProductManager struct {
	sync.RWMutex
	cache map[string]codec.Product
}

func (p *redisProductManager) Id() string {
	return "redis"
}

func (m *redisProductManager) Get(productId string) codec.Product {
	product, ok := m.cache[productId]
	if ok {
		return product
	}
	if product == nil {
		m.Lock()
		defer m.Unlock()
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
		defer cancel()
		rdb := codec.GetRedisClient()
		data, err := rdb.HGetAll(ctx, "goiot:product:"+productId).Result()
		if err != nil {
			logs.Error(err)
		}
		if data == nil {
			m.cache[productId] = nil
		}
		config := map[string]string{}
		err = json.Unmarshal([]byte(data["config"]), &config)
		if err != nil {
			logs.Error(err)
		}
		produ, err := codec.NewProduct(data["id"], config, data["storePolicy"], data["tslData"])
		if err != nil {
			logs.Error(err)
		} else {
			m.cache[productId] = produ
			return produ
		}
	}
	return nil
}

func (m *redisProductManager) Put(product codec.Product) {
	if product == nil {
		panic("product not be nil")
	}
	if len(product.GetId()) == 0 {
		panic("product id must be present")
	}
	p := product.(*codec.DefaultProdeuct)
	byt, _ := json.Marshal(p.Config)
	data := map[string]string{
		"id":          p.Id,
		"config":      string(byt),
		"storePolicy": p.StorePolicy,
		"tslData":     p.TslData.Text,
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	rdb := codec.GetRedisClient()
	err := rdb.HSet(ctx, "goiot:product:"+p.Id, data).Err()
	if err != nil {
		logs.Error(err)
	}
	m.cache[product.GetId()] = product
}
