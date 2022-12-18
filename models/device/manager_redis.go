package models

import (
	"context"
	"encoding/json"
	"fmt"
	"go-iot/codec"
	"go-iot/codec/util"
	"sync"
	"time"

	"github.com/beego/beego/v2/core/logs"
)

func init() {
	codec.RegDeviceManager(&redisDeviceManager{cache: make(map[string]*codec.Device)})
	codec.RegProductManager(&redisProductManager{cache: make(map[string]*codec.Product)})
}

// redisDeviceManager
type redisDeviceManager struct {
	sync.RWMutex
	cache map[string]*codec.Device
}

func (p *redisDeviceManager) Id() string {
	return "redis"
}

func (m *redisDeviceManager) Get(deviceId string) *codec.Device {
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
		data, err := rdb.HGetAll(ctx, "goiot:device:"+deviceId).Result()
		if err != nil {
			logs.Error(err)
		}
		if len(data) == 0 {
			m.cache[deviceId] = nil
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
		dev := &codec.Device{
			Id:        data["id"],
			ProductId: data["productId"],
			CreateId:  createId,
			Config:    config,
			Data:      dat,
		}
		m.cache[deviceId] = dev
		return dev
	}
	return nil
}

func (m *redisDeviceManager) Put(device *codec.Device) {
	p := device
	byt, _ := json.Marshal(p.Config)
	dat, _ := json.Marshal(p.Data)
	data := map[string]string{
		"id":        p.Id,
		"productId": p.ProductId,
		"createId":  fmt.Sprintf("%v", p.CreateId),
		"config":    string(byt),
		"data":      string(dat),
	}
	rdb := codec.GetRedisClient()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	rdb.HSet(ctx, "goiot:device:"+p.Id, data).Err()
	m.cache[device.GetId()] = device
}

type redisProductManager struct {
	sync.RWMutex
	cache map[string]*codec.Product
}

func (p *redisProductManager) Id() string {
	return "redis"
}

func (m *redisProductManager) Get(productId string) *codec.Product {
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
		if len(data) == 0 {
			m.cache[productId] = nil
			return nil
		}
		config := map[string]string{}
		if str, ok := data["config"]; ok {
			err = json.Unmarshal([]byte(str), &config)
			if err != nil {
				logs.Error("device config parse error:", err)
			}
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

func (m *redisProductManager) Put(product *codec.Product) {
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
	rdb := codec.GetRedisClient()
	err := rdb.HSet(ctx, "goiot:product:"+p.Id, data).Err()
	if err != nil {
		logs.Error(err)
	}
	m.cache[product.GetId()] = product
}
