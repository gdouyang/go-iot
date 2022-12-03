package codec

import (
	"context"
	"errors"
	"fmt"
	"go-iot/codec/msg"
	"sync"
	"time"
)

// 进行功能调用
func DoCmdInvoke(productId string, message msg.FuncInvoke) error {
	session := sessionManager.Get(message.DeviceId)
	if session == nil {
		return fmt.Errorf("device %s is offline", message.DeviceId)
	}
	codec := GetCodec(productId)
	if codec == nil {
		return fmt.Errorf("codec %s of product not found", productId)
	}
	product := GetProductManager().Get(productId)
	if product == nil {
		return fmt.Errorf("product %s not found", productId)
	}
	tslF, ok := product.GetTslFunction()[message.FunctionId]
	if !ok {
		return fmt.Errorf("function %s of tsl not found", message.FunctionId)
	}
	if tslF.Async {
		go func() {
			codec.OnInvoke(&FuncInvokeContext{
				deviceId:  message.DeviceId,
				productId: productId,
				session:   session,
				message:   message,
			})
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
			err := codec.OnInvoke(&FuncInvokeContext{
				deviceId:  message.DeviceId,
				productId: productId,
				session:   session,
				message:   message,
			})
			if nil != err {
				replyMap.reply(message.DeviceId, err)
			}
		}(ctx)
		select {
		case <-ctx.Done():
			return errors.New("invoke timeout")
		case err := <-message.Replay:
			return err
		}
	}
}

// 功能调用
type FuncInvokeContext struct {
	message   msg.FuncInvoke
	session   Session
	deviceId  string
	productId string
}

func (ctx *FuncInvokeContext) GetMessage() interface{} {
	return ctx.message
}
func (ctx *FuncInvokeContext) GetSession() Session {
	return ctx.session
}

// 获取设备操作
func (ctx *FuncInvokeContext) GetDevice() Device {
	return GetDeviceManager().Get(ctx.deviceId)
}

// 获取产品操作
func (ctx *FuncInvokeContext) GetProduct() Product {
	return GetProductManager().Get(ctx.productId)
}

func (ctx *FuncInvokeContext) ReplyOk() {
	replyMap.reply(ctx.deviceId, nil)
}

func (ctx *FuncInvokeContext) ReplyFail(resp string) {
	replyMap.reply(ctx.deviceId, errors.New(resp))
}

// cmd invoke reply
var replyMap = &funcInvokeReply{}

type funcInvokeReply struct {
	m sync.Map
}

type reply struct {
	time   int64
	expire int64
	cmd    *msg.FuncInvoke
}

func (r *funcInvokeReply) addReply(i *msg.FuncInvoke, exprie time.Duration) error {
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
