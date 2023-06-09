package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go-iot/pkg/boot"
	"go-iot/pkg/cluster"
	"go-iot/pkg/core/common"
	"go-iot/pkg/redis"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	logs "go-iot/pkg/logger"
)

func init() {
	boot.AddStartLinstener(func() {
		go listenerCluster()
	})
}

func listenerCluster() {
	if cluster.Enabled() {
		for redisMsg := range redis.Sub("go:cluster:cmdinvoke") {
			payload := redisMsg.Payload
			var message common.FuncInvoke
			json.Unmarshal([]byte(payload), &message)
			if message.ClusterId == cluster.GetClusterId() {
				go DoCmdInvoke(message)
			}
		}
	}
}

func DoCmdInvokeCluster(message common.FuncInvoke) {
	if cluster.Enabled() {
		device := GetDevice(message.DeviceId)
		if device.ClusterId != cluster.GetClusterId() {
			message.ClusterId = device.ClusterId
			data, _ := json.Marshal(message)
			redis.Pub("go:cluster:cmdinvoke", data)
		}
	} else {
		DoCmdInvoke(message)
	}
}

// 进行功能调用
func DoCmdInvoke(message common.FuncInvoke) error {
	session := GetSession(message.DeviceId)
	if session == nil {
		return fmt.Errorf("device %s is offline", message.DeviceId)
	}
	device := GetDevice(message.DeviceId)
	productId := device.ProductId
	codec := GetCodec(productId)
	if codec == nil {
		return fmt.Errorf("core %s of product not found", productId)
	}
	product := GetProduct(productId)
	if product == nil {
		return fmt.Errorf("product %s not found, make sure deployed", productId)
	}
	tslF := product.GetTsl().FunctionsMap()
	if len(tslF) == 0 {
		return fmt.Errorf("product [%s] have no functions", productId)
	}
	function, ok := tslF[message.FunctionId]
	if !ok {
		return fmt.Errorf("function [%s] of tsl not found", message.FunctionId)
	}
	if product != nil {
		b, _ := json.Marshal(message)
		product.GetTimeSeries().SaveLogs(product, LogData{DeviceId: message.DeviceId, Type: "call", Content: string(b)})
	}
	invokeContext := FuncInvokeContext{
		BaseContext: BaseContext{
			DeviceId:  message.DeviceId,
			ProductId: productId,
			Session:   session,
		},
		message: message,
	}
	if function.Async {
		go func() {
			codec.OnInvoke(invokeContext)
		}()
		return nil
	} else {
		timeout := (time.Second * 10)
		err := replyMap.addReply(&message, timeout)
		if err != nil {
			return err
		}
		// timeout of invoke
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		message.Replay = make(chan error)
		go func(ctx context.Context) {
			err := codec.OnInvoke(invokeContext)
			if nil != err {
				replyMap.reply(message.DeviceId, err)
			}
		}(ctx)
		select {
		case <-ctx.Done():
			err = errors.New("invoke timeout")
			replyLog(product, message, err.Error())
			return err
		case err := <-message.Replay:
			if err != nil {
				replyLog(product, message, err.Error())
			} else {
				replyLog(product, message, "")
			}
			return err
		}
	}
}

func replyLog(product *Product, message common.FuncInvoke, reply string) {
	if product != nil {
		aligs := struct {
			common.FuncInvoke
			Reply string `json:"reply"`
		}{
			FuncInvoke: message,
			Reply:      reply,
		}
		b, _ := json.Marshal(aligs)
		product.GetTimeSeries().SaveLogs(product, LogData{DeviceId: message.DeviceId, Type: "reply", Content: string(b)})
	}
}

// 功能调用
type FuncInvokeContext struct {
	BaseContext
	message common.FuncInvoke
}

func (ctx *FuncInvokeContext) DeviceOnline(deviceId string) {
}

func (ctx *FuncInvokeContext) GetMessage() interface{} {
	return ctx.message
}

// cmd invoke reply
var replyMap = &funcInvokeReply{}

type funcInvokeReply struct {
	m sync.Map
}

type reply struct {
	time   int64
	expire int64
	cmd    *common.FuncInvoke
}

func (r *funcInvokeReply) addReply(i *common.FuncInvoke, exprie time.Duration) error {
	val, ok := r.m.Load(i.DeviceId)
	now := time.Now().UnixMilli()
	if ok {
		v := val.(*reply)
		if v.expire > now {
			return fmt.Errorf("invoke [%s] is in process, please try later", i.FunctionId)
		}
	}
	r.m.Store(i.DeviceId, &reply{
		time:   now,
		expire: now + exprie.Milliseconds(),
		cmd:    i,
	})
	return nil
}

func (r *funcInvokeReply) reply(deviceId string, resp error) {
	val, ok := r.m.Load(deviceId)
	if ok {
		v := val.(*reply)
		v.cmd.Replay <- resp
	}
	r.m.Delete(deviceId)
}

// http request func for http network
func HttpRequest(config map[string]interface{}) map[string]interface{} {
	result := map[string]interface{}{}
	path := config["url"]
	u, err := url.ParseRequestURI(fmt.Sprintf("%v", path))
	if err != nil {
		logs.Errorf(err.Error())
		result["status"] = 400
		result["message"] = err.Error()
		return result
	}
	method := fmt.Sprintf("%v", config["method"])
	client := http.Client{Timeout: time.Second * 3}
	var req *http.Request = &http.Request{
		Method: strings.ToUpper(method),
		URL:    u,
		Header: map[string][]string{},
	}
	if v, ok := config["header"]; ok {
		h, ok := v.(map[string]interface{})
		if !ok {
			logs.Warnf("header is not object: %v", v)
			h = map[string]interface{}{}
		}
		for key, value := range h {
			req.Header.Add(key, fmt.Sprintf("%v", value))
		}
	}
	if strings.ToLower(method) == "post" && (len(req.Header.Get("Content-Type")) == 0 || len(req.Header.Get("content-type")) == 0) {
		req.Header.Add("Content-Type", "application/json; charset=utf-8")
	}
	if data, ok := config["data"]; ok {
		if body, ok := data.(map[string]interface{}); ok {
			b, err := json.Marshal(body)
			if err != nil {
				logs.Errorf("http data parse error: %v", err)
				result["status"] = 400
				result["message"] = err.Error()
				return result
			}
			req.Body = io.NopCloser(strings.NewReader(string(b)))
		} else {
			req.Body = io.NopCloser(strings.NewReader(fmt.Sprintf("%v", data)))
		}
	}
	resp, err := client.Do(req)
	if err != nil {
		logs.Errorf(err.Error())
		result["status"] = resp.StatusCode
		result["message"] = err.Error()
		return result
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		logs.Errorf(err.Error())
		result["status"] = 400
		result["message"] = err.Error()
		return result
	}
	header := map[string]string{}
	if resp.Header != nil {
		for key := range resp.Header {
			header[key] = resp.Header.Get(key)
		}
	}
	result["data"] = string(b)
	result["status"] = resp.StatusCode
	result["header"] = header
	return result
}
