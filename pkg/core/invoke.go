package core

import (
	"context"
	"encoding/json"
	"fmt"
	"go-iot/pkg/boot"
	"go-iot/pkg/cluster"
	"go-iot/pkg/core/common"
	"go-iot/pkg/redis"
	"sync"
	"time"

	"github.com/google/uuid"
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
func DoCmdInvoke(message common.FuncInvoke) *common.Err {
	session := GetSession(message.DeviceId)
	if session == nil {
		return common.NewErr400("设备已离线")
	}
	device := GetDevice(message.DeviceId)
	productId := device.ProductId
	product := GetProduct(productId)
	if product == nil {
		return common.NewErr400(fmt.Sprintf("产品[%s]不存在，请确产品已发布", productId))
	}
	codec := GetCodec(productId)
	if codec == nil {
		return common.NewErr400(fmt.Sprintf("产品[%s]没有配置编解码", productId))
	}
	tslF := product.GetTsl().FunctionsMap()
	if len(tslF) == 0 {
		return common.NewErr400(fmt.Sprintf("产品[%s]没有配置功能", productId))
	}
	function, ok := tslF[message.FunctionId]
	if !ok {
		return common.NewErr400(fmt.Sprintf("功能[%s]不存在", message.FunctionId))
	}
	if len(message.TraceId) == 0 {
		message.TraceId = uuid.NewString()
	}
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
	if async {
		go func() {
			codec.OnInvoke(invokeContext)
		}()
		return nil
	} else {
		timeout := (time.Second * 10)
		if message.Timeout > 0 {
			timeout = time.Duration(time.Second * 10)
		}
		err := replyMap.addReply(&message, timeout)
		if err != nil {
			return common.NewErr500(err.Error())
		}
		// timeout of invoke
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		message.Replay = make(chan *common.FuncInvokeReply)
		go func(ctx context.Context) {
			err := codec.OnInvoke(invokeContext)
			if nil != err {
				message.Replay <- &common.FuncInvokeReply{Success: false, Msg: err.Error()}
			}
		}(ctx)
		select {
		case <-ctx.Done():
			err = fmt.Errorf("功能[%s]调用超时", message.FunctionId)
			replyLogSync(product, message, &common.FuncInvokeReply{Success: false, Msg: err.Error()})
			return common.NewErr504(err.Error())
		case resp := <-message.Replay:
			if resp != nil && !resp.Success {
				// 失败
				replyLogSync(product, message, resp)
				if len(resp.Msg) > 0 {
					return common.NewErr504(resp.Msg)
				}
				return common.NewErr504("请求失败")
			}
			// 成功
			replyLogSync(product, message, &common.FuncInvokeReply{Success: true})
			return nil
		}
	}
}

// 同步命令回复
func replyLogSync(product *Product, message common.FuncInvoke, reply *common.FuncInvokeReply) {
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
func replyLogAsync(product *Product, deviceId string, reply *common.FuncInvokeReply) {
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
	message common.FuncInvoke
}

func (ctx *FuncInvokeContext) DeviceOnline(deviceId string) {
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
	cmd    *common.FuncInvoke
}

func (r *funcInvokeReplyManager) addReply(i *common.FuncInvoke, exprie time.Duration) error {
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

func (r *funcInvokeReplyManager) reply(deviceId string, resp *common.FuncInvokeReply) {
	val, ok := r.m.Load(deviceId)
	if ok {
		v := val.(*reply)
		v.cmd.Replay <- resp
	}
	r.m.Delete(deviceId)
}
