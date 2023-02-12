package modbus

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"go-iot/pkg/core"

	"github.com/beego/beego/v2/core/logs"
)

type context struct {
	core.BaseContext
	Data []byte
}

func (ctx *context) GetMessage() interface{} {
	return ctx.Data
}

func (ctx *context) MsgToString() string {
	return string(bytes.Trim(ctx.Data, string(rune(0))))
}

func (ctx *context) MsgToHexStr() string {
	return hex.EncodeToString(ctx.Data)
}

func (ctx *context) MsgToUint16() uint16 {
	return binary.BigEndian.Uint16(ctx.Data)
}

func (ctx *context) MsgToUint32() uint32 {
	return binary.BigEndian.Uint32(swap32BitDataBytes(ctx.Data, false, false))
}

func (ctx *context) MsgToUint64() uint64 {
	return binary.BigEndian.Uint64(ctx.Data)
}

func (ctx *context) MsgToInt16() int16 {
	return int16(ctx.MsgToUint16())
}
func (ctx *context) MsgToInt32() int32 {
	return int32(ctx.MsgToUint32())
}
func (ctx *context) MsgToInt64() int64 {
	return int64(ctx.MsgToUint64())
}
func (ctx *context) MsgToBool() bool {
	return (ctx.Data[0] & 1) > 0
}

type modbusInvokeContext struct {
	core.FuncInvokeContext
}

func (ctx *modbusInvokeContext) Int16ToData(val int16) string {
	dataBytes, err := getBinaryData(int16(val))
	if err != nil {
		logs.Warn(err)
	}
	return hex.EncodeToString(dataBytes)
}
func (ctx *modbusInvokeContext) FloatToInt16Data(val float64) string {
	dataBytes, err := getBinaryData(int16(val))
	if err != nil {
		logs.Warn(err)
	}
	return hex.EncodeToString(dataBytes)
}

func (ctx *modbusInvokeContext) FloatToUint16Data(val float64) string {
	dataBytes, err := getBinaryData(uint16(val))
	if err != nil {
		logs.Warn(err)
	}
	return hex.EncodeToString(dataBytes)
}

func getBinaryData(val interface{}) (dataBytes []byte, err error) {
	buf := new(bytes.Buffer)
	err = binary.Write(buf, binary.BigEndian, val)
	if err != nil {
		return dataBytes, err
	}
	dataBytes = buf.Bytes()
	return dataBytes, err
}
