package util

import (
	"bytes"
	"encoding/binary"

	"github.com/beego/beego/v2/core/logs"
)

func BigEndianUnit16(bytes []byte) uint16 {
	return binary.BigEndian.Uint16(bytes)
}
func BigEndianUnit32(bytes []byte) uint32 {
	return binary.BigEndian.Uint32(bytes)
}
func BigEndianUnit64(bytes []byte) uint64 {
	return binary.BigEndian.Uint64(bytes)
}

func BigEndianFloatToInt16Data(val float64) []byte {
	dataBytes, err := getBinaryData(int16(val))
	if err != nil {
		logs.Warn(err)
	}
	return dataBytes
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
