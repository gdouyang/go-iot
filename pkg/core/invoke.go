package core

import (
	"context"
	"encoding/json"
	"fmt"
	"go-iot/pkg/common"
	"go-iot/pkg/tsl"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	// OTA升级
	OTA_UPDATE = "ota-update"
)

// 进行功能调用
func DoCmdInvoke(message FuncInvoke) *common.Err {
	return doCmdInvoke(message, false)
}

// 进行功能调用，缓存离线命令, 成功返回true, 失败返回false
func DoCmdInvokeOffline(message FuncInvoke) (bool, *common.Err) {
	err := doCmdInvoke(message, true)
	if err != nil && err.Code != 200 {
		return false, err
	}
	return true, err
}

// 进行功能调用，cache为true时，缓存离线命令
func doCmdInvoke(message FuncInvoke, cache bool) *common.Err {
	device := GetDevice(message.DeviceId)
	if device == nil {
		return common.NewErr400(fmt.Sprintf("设备[%s]不存在，请确认设备已注册并激活", message.DeviceId))
	}
	productId := device.ProductId
	product := GetProduct(productId)
	if product == nil {
		return common.NewErr400(fmt.Sprintf("产品[%s]不存在，请确产品已发布", productId))
	}
	codec := GetCodec(productId)
	if codec == nil {
		return common.NewErr400(fmt.Sprintf("产品[%s]没有配置编解码", productId))
	}
	var function tsl.Function
	var ok bool
	if message.FunctionId == OTA_UPDATE {
		function = tsl.Function{
			Id:    OTA_UPDATE,
			Name:  "OTA升级",
			Async: false,
		}
	} else {
		tslF := product.GetTsl().FunctionsMap()
		if len(tslF) == 0 {
			return common.NewErr400(fmt.Sprintf("产品[%s]没有配置功能", productId))
		}
		function, ok = tslF[message.FunctionId]
		if !ok {
			return common.NewErr400(fmt.Sprintf("功能[%s]不存在", message.FunctionId))
		}
	}
	if len(message.TraceId) == 0 {
		message.TraceId = strings.ReplaceAll(uuid.NewString(), "-", "")
	}
	timeout := time.Duration(time.Second * 10)
	if message.Timeout > 0 {
		timeout = time.Duration(message.Timeout) * time.Second
	}
	state := GetDeviceState(message.DeviceId, productId)
	// 对于http协议来说，设备离线时，也可以调用功能
	session := GetSession(message.DeviceId)
	// 缓存离线命令
	if cache {
		isDisconnect := IsDeviceDisconnect(message.DeviceId)
		if OFFLINE == state || isDisconnect {
			return cacheOfflineCommand(message)
		}
	}
	// 设备已离线，返回错误
	if OFFLINE == state {
		return common.NewErr400("设备已离线")
	}
	// 设备已连接，发送命令
	b, _ := json.Marshal(message)
	product.GetTimeSeries().SaveLogs(product,
		LogData{
			Type:     "call",
			TraceId:  message.TraceId,
			DeviceId: message.DeviceId,
			Content:  string(b),
		},
	)
	invokeContext := FuncInvokeContext{
		BaseContext: BaseContext{
			DeviceId:  message.DeviceId,
			ProductId: productId,
			Session:   session,
		},
		message: message,
	}
	async := message.Async == "true" || function.Async
	if message.Async == "false" {
		async = false
	}
	if async {
		go func() {
			codec.OnInvoke(invokeContext)
		}()
		return nil
	} else {
		err := replyMap.addReply(&message, timeout)
		if err != nil {
			return common.NewErr500(err.Error())
		}
		// timeout of invoke
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		message.Replay = make(chan *FuncInvokeReply)
		go func(ctx context.Context) {
			err = codec.OnInvoke(invokeContext)
			if nil != err {
				message.Replay <- &FuncInvokeReply{Success: false, Msg: err.Error()}
			}
		}(ctx)
		select {
		case <-ctx.Done():
			err = fmt.Errorf("功能[%s]调用超时", message.FunctionId)
			replyLogSync(product, message, &FuncInvokeReply{Success: false, Msg: err.Error()})
			return common.NewErr504(err.Error())
		case resp := <-message.Replay:
			if resp != nil && !resp.Success {
				replyMap.deleteReply(message.DeviceId)
				// 失败
				replyLogSync(product, message, resp)
				if len(resp.Msg) > 0 {
					return common.NewErr504(resp.Msg)
				}
				return common.NewErr504("请求失败")
			}
			// 成功
			replyLogSync(product, message, &FuncInvokeReply{Success: true})
			return nil
		}
	}
}

// 同步命令回复
func replyLogSync(product *Product, message FuncInvoke, reply *FuncInvokeReply) {
	if product != nil {
		b, _ := json.Marshal(reply)
		product.GetTimeSeries().SaveLogs(product,
			LogData{
				Type:     "reply",
				DeviceId: message.DeviceId,
				TraceId:  message.TraceId,
				Content:  string(b),
			},
		)
	}
}

// 异步命令回复
func replyLogAsync(product *Product, deviceId string, reply *FuncInvokeReply) {
	if product != nil && reply != nil {
		b, _ := json.Marshal(reply)
		product.GetTimeSeries().SaveLogs(product,
			LogData{
				Type:     "reply",
				DeviceId: deviceId,
				TraceId:  reply.TraceId,
				Content:  string(b),
			},
		)
	}
}

// 功能调用
type FuncInvokeContext struct {
	BaseContext
	message FuncInvoke
}

// DeviceOnline 功能调用上下文不允许通过此方法上线设备。
func (ctx *FuncInvokeContext) DeviceOnline(deviceId string) error {
	return nil
}

func (ctx *FuncInvokeContext) GetMessage() interface{} {
	return ctx.message
}

// cmd invoke reply
var replyMap = &funcInvokeReplyManager{}

type funcInvokeReplyManager struct {
	m sync.Map
}

type reply struct {
	time   int64
	expire int64
	cmd    *FuncInvoke
}

func (r *funcInvokeReplyManager) addReply(i *FuncInvoke, exprie time.Duration) error {
	val, ok := r.m.Load(i.DeviceId)
	now := time.Now().UnixMilli()
	if ok {
		v := val.(*reply)
		if v.expire > now {
			return fmt.Errorf("功能[%s]正在执行,请稍后再试", i.FunctionId)
		}
	}
	r.m.Store(i.DeviceId, &reply{
		time:   now,
		expire: now + exprie.Milliseconds(),
		cmd:    i,
	})
	return nil
}

func (r *funcInvokeReplyManager) reply(deviceId string, resp *FuncInvokeReply) {
	val, ok := r.m.Load(deviceId)
	if ok {
		v := val.(*reply)
		v.cmd.Replay <- resp
	}
	r.m.Delete(deviceId)
}

func (r *funcInvokeReplyManager) deleteReply(deviceId string) {
	r.m.Delete(deviceId)
}

// 缓存离线命令（经 OfflineCommandQueue，实现由 App 注入 Redis/Memory）
func cacheOfflineCommand(message FuncInvoke) *common.Err {
	return defaultOfflineQueue.Enqueue(message)
}

// 发送离线命令
func sendOfflineCommands(deviceId string) {
	cmds := defaultOfflineQueue.TakeAll(deviceId)
	if len(cmds) == 0 {
		return
	}
	// 异步执行，避免阻塞上线路径
	go func() {
		for _, message := range cmds {
			message.Async = "false"
			DoCmdInvoke(message)
		}
	}()
}
