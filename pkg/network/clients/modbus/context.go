package modbus

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"go-iot/pkg/core"

	logs "go-iot/pkg/logger"
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

func hexEncodeBinary(val any) string {
	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.BigEndian, val); err != nil {
		logs.Warnf(err.Error())
		return ""
	}
	return hex.EncodeToString(buf.Bytes())
}
