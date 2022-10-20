package codec

import (
	"context"
	"errors"
	"go-iot/codec/msg"
	"time"
)

// 进行功能调用
func DoCmdInvoke(productId string, message msg.FuncInvoke) error {
	session := sessionManager.Get(message.DeviceId)
	if session == nil {
		return errors.New("device is offline")
	}
	codec := GetCodec(productId)
	if codec == nil {
		return errors.New("not found codec")
	}
	// timeout of invoke
	ctx, cancel := context.WithTimeout(context.Background(), (time.Second * 10))
	defer cancel()

	result := make(chan error)
	go func(ctx context.Context) {
		err := codec.OnInvoke(&FuncInvokeContext{
			deviceId:  message.DeviceId,
			productId: productId,
			session:   session,
			message:   message,
		})
		result <- err
	}(ctx)

	select {
	case <-ctx.Done():
		return errors.New("timeout")
	case err := <-result:
		return err
	}
}

// 功能调用
type FuncInvokeContext struct {
	message   interface{}
	session   Session
	deviceId  string
	productId string
	// functionId string
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
