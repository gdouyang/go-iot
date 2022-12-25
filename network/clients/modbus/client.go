package modbus

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/beego/beego/v2/core/logs"
)

// DeviceClient is a interface for modbus client lib to implementation
// It's responsibility are handle connection, read data bytes value and write data bytes value
type DeviceClient interface {
	OpenConnection() error
	GetValue(commandInfo interface{}) ([]byte, error)
	SetValue(commandInfo interface{}, value []byte) error
	CloseConnection() error
}

// CommandInfo is command info
type CommandInfo struct {
	PrimaryTable    string
	StartingAddress uint16
	ValueType       string
	// how many register need to read
	Length     uint16
	IsByteSwap bool
	IsWordSwap bool
	RawType    string
}

// CommandRequest is the struct for requesting a command to ProtocolDrivers
type CommandRequest struct {
	// Attributes is a key/value map to represent the attributes of the Device Resource
	Attributes map[string]interface{}
	// Type is the data type of the Device Resource
	Type string
}

type CommandValue struct {
	// Type indicates what type of value was returned from the ProtocolDriver instance in
	// response to HandleCommand being called to handle a single ResourceOperation.
	Type string
	// Value holds value returned by a ProtocolDriver instance.
	// The value can be converted to its native type by referring to ValueType.
	Value interface{}
	// Origin is an int64 value which indicates the time the reading
	// contained in the CommandValue was read by the ProtocolDriver
	// instance.
	Origin int64
	// Tags allows device service to add custom information to the Event in order to
	// help identify its origin or otherwise label it before it is send to north side.
	Tags map[string]string
}

// ValueToString returns the string format of the value.
func (cv *CommandValue) ValueToString() string {
	if cv.Type == "Binary" {
		binaryValue := cv.Value.([]byte)
		return fmt.Sprintf("Binary: [%v...]", string(binaryValue[:20]))
	}
	return fmt.Sprintf("%v", cv.Value)
}
func (cv *CommandValue) Float32Value() (float32, error) {
	var value float32
	if cv.Type != ValueTypeFloat64 {
		errMsg := fmt.Errorf("cannot convert CommandValue of %s to %s", cv.Type, ValueTypeFloat64)
		return value, errMsg
	}
	value, ok := cv.Value.(float32)
	if !ok {
		errMsg := fmt.Errorf("failed to transfrom %v to %T", cv.Value, value)
		return value, errMsg
	}
	return value, nil
}

// Float64Value returns the value in float64 data type, and returns error if the Type is not Float64.
func (cv *CommandValue) Float64Value() (float64, error) {
	var value float64
	if cv.Type != ValueTypeFloat64 {
		errMsg := fmt.Errorf("cannot convert CommandValue of %s to %s", cv.Type, ValueTypeFloat64)
		return value, errMsg
	}
	value, ok := cv.Value.(float64)
	if !ok {
		errMsg := fmt.Errorf("failed to transfrom %v to %T", cv.Value, value)
		return value, errMsg
	}
	return value, nil
}
func createCommandInfo(req *CommandRequest) (*CommandInfo, error) {
	if _, ok := req.Attributes[PRIMARY_TABLE]; !ok {
		return nil, fmt.Errorf("attribute %s not exists", PRIMARY_TABLE)
	}
	primaryTable := fmt.Sprintf("%v", req.Attributes[PRIMARY_TABLE])
	primaryTable = strings.ToUpper(primaryTable)

	if _, ok := req.Attributes[STARTING_ADDRESS]; !ok {
		return nil, fmt.Errorf("attribute %s not exists", STARTING_ADDRESS)
	}
	startingAddress, err := castStartingAddress(req.Attributes[STARTING_ADDRESS])
	if err != nil {
		return nil, fmt.Errorf("fail to cast %s", STARTING_ADDRESS)
	}

	var rawType = req.Type
	if _, ok := req.Attributes[RAW_TYPE]; ok {
		rawType = fmt.Sprintf("%v", req.Attributes[RAW_TYPE])
		rawType, err = normalizeRawType(rawType)
		if err != nil {
			return nil, err
		}
	}
	var length uint16
	if req.Type == "String" {
		length, err = castStartingAddress(req.Attributes[STRING_REGISTER_SIZE])
		if err != nil {
			return nil, err
		} else if (length > 123) || (length < 1) {
			return nil, fmt.Errorf("register size should be within the range of 1~123, get %v. ", length)
		}
	} else {
		length = calculateAddressLength(primaryTable, rawType)
	}

	var isByteSwap = false
	if _, ok := req.Attributes[IS_BYTE_SWAP]; ok {
		isByteSwap, err = castSwapAttribute(req.Attributes[IS_BYTE_SWAP])
		if err != nil {
			return nil, fmt.Errorf("fail to cast %s", IS_BYTE_SWAP)
		}
	}

	var isWordSwap = false
	if _, ok := req.Attributes[IS_WORD_SWAP]; ok {
		isWordSwap, err = castSwapAttribute(req.Attributes[IS_WORD_SWAP])
		if err != nil {
			return nil, fmt.Errorf("fail to cast %s", IS_WORD_SWAP)
		}
	}

	return &CommandInfo{
		PrimaryTable:    primaryTable,
		StartingAddress: startingAddress,
		ValueType:       req.Type,
		Length:          length,
		IsByteSwap:      isByteSwap,
		IsWordSwap:      isWordSwap,
		RawType:         rawType,
	}, nil
}

func calculateAddressLength(primaryTable string, valueType string) uint16 {
	var primaryTableBit = PrimaryTableBitCountMap[primaryTable]
	var valueTypeBitCount = ValueTypeBitCountMap[valueType]

	var length = valueTypeBitCount / primaryTableBit
	if length < 1 {
		length = 1
	}

	return length
}

// TransformDataBytesToResult is used to transform the device's binary data to the specified value type as the actual result.
func TransformDataBytesToResult(req *CommandRequest, dataBytes []byte, commandInfo *CommandInfo) (*CommandValue, error) {
	var err error
	var res interface{}
	var result = &CommandValue{}

	switch commandInfo.ValueType {
	case ValueTypeUint16:
		res = binary.BigEndian.Uint16(dataBytes)
	case ValueTypeUint32:
		res = binary.BigEndian.Uint32(swap32BitDataBytes(dataBytes, commandInfo.IsByteSwap, commandInfo.IsWordSwap))
	case ValueTypeUint64:
		res = binary.BigEndian.Uint64(dataBytes)
	case ValueTypeInt16:
		res = int16(binary.BigEndian.Uint16(dataBytes))
	case ValueTypeInt32:
		res = int32(binary.BigEndian.Uint32(swap32BitDataBytes(dataBytes, commandInfo.IsByteSwap, commandInfo.IsWordSwap)))
	case ValueTypeInt64:
		res = int64(binary.BigEndian.Uint64(dataBytes))
	case ValueTypeFloat32:
		switch commandInfo.RawType {
		case ValueTypeFloat32:
			raw := binary.BigEndian.Uint32(swap32BitDataBytes(dataBytes, commandInfo.IsByteSwap, commandInfo.IsWordSwap))
			res = math.Float32frombits(raw)
		case ValueTypeInt16:
			raw := int16(binary.BigEndian.Uint16(dataBytes))
			res = float32(raw)
			logs.Debug("According to the rawType %s and the value type %s, convert integer %d to float %v ", INT16, FLOAT32, res, result.ValueToString())
		case ValueTypeUint16:
			raw := binary.BigEndian.Uint16(dataBytes)
			res = float32(raw)
			logs.Debug("According to the rawType %s and the value type %s, convert integer %d to float %v ", UINT16, FLOAT32, res, result.ValueToString())
		}
	case ValueTypeFloat64:
		switch commandInfo.RawType {
		case ValueTypeFloat64:
			raw := binary.BigEndian.Uint64(dataBytes)
			res = math.Float64frombits(raw)
		case ValueTypeInt16:
			raw := int16(binary.BigEndian.Uint16(dataBytes))
			res = float64(raw)
			logs.Debug("According to the rawType %s and the value type %s, convert integer %d to float %v ", INT16, FLOAT64, res, result.ValueToString())
		case ValueTypeUint16:
			raw := binary.BigEndian.Uint16(dataBytes)
			res = float64(raw)
			logs.Debug("According to the rawType %s and the value type %s, convert integer %d to float %v ", UINT16, FLOAT64, res, result.ValueToString())
		}
	case ValueTypeBool:
		res = false
		// to find the 1st bit of the dataBytes by mask it with 2^0 = 1 (00000001)
		if (dataBytes[0] & 1) > 0 {
			res = true
		}
	case ValueTypeString:
		res = string(bytes.Trim(dataBytes, string(rune(0))))
	default:
		return nil, fmt.Errorf("return result fail, none supported value type: %v", commandInfo.ValueType)
	}

	result = &CommandValue{Type: commandInfo.ValueType, Value: res, Tags: make(map[string]string)}
	if err != nil {
		return nil, err
	}
	result.Origin = time.Now().UnixNano()

	logs.Debug("Transfer dataBytes to CommandValue(%v) successful.", result.ValueToString())
	return result, nil
}

// TransformCommandValueToDataBytes transforms the reading value to binary data which is used to transfer data via Modbus protocol.
func TransformCommandValueToDataBytes(commandInfo *CommandInfo, value *CommandValue) ([]byte, error) {
	var err error
	var byteCount = calculateByteCount(commandInfo)
	var dataBytes []byte
	buf := new(bytes.Buffer)
	if commandInfo.ValueType != "String" {
		err = binary.Write(buf, binary.BigEndian, value.Value)
		if err != nil {
			return nil, fmt.Errorf("failed to transform %v to []byte", value.Value)
		}

		numericValue := buf.Bytes()
		var maxSize = uint16(len(numericValue))
		dataBytes = numericValue[maxSize-byteCount : maxSize]
	}

	_, ok := ValueTypeBitCountMap[commandInfo.ValueType]
	if !ok {
		err = fmt.Errorf("none supported value type : %v ", commandInfo.ValueType)
		return dataBytes, err
	}

	if commandInfo.ValueType == ValueTypeUint32 || commandInfo.ValueType == ValueTypeInt32 || commandInfo.ValueType == ValueTypeFloat32 {
		dataBytes = swap32BitDataBytes(dataBytes, commandInfo.IsByteSwap, commandInfo.IsWordSwap)
	}

	// Cast value according to the rawType, this feature converts float value to integer 32bit value
	if commandInfo.ValueType == ValueTypeFloat32 {
		val, edgexErr := value.Float32Value()
		if edgexErr != nil {
			return dataBytes, edgexErr
		}
		if commandInfo.RawType == ValueTypeInt16 {
			dataBytes, err = getBinaryData(int16(val))
			if err != nil {
				return dataBytes, err
			}
		} else if commandInfo.RawType == ValueTypeUint16 {
			dataBytes, err = getBinaryData(uint16(val))
			if err != nil {
				return dataBytes, err
			}
		}
	} else if commandInfo.ValueType == ValueTypeFloat64 {
		val, edgexErr := value.Float64Value()
		if edgexErr != nil {
			return dataBytes, edgexErr
		}
		if commandInfo.RawType == ValueTypeInt16 {
			dataBytes, err = getBinaryData(int16(val))
			if err != nil {
				return dataBytes, err
			}
		} else if commandInfo.RawType == ValueTypeUint16 {
			dataBytes, err = getBinaryData(uint16(val))
			if err != nil {
				return dataBytes, err
			}
		}
	} else if commandInfo.ValueType == ValueTypeString {
		// Cast value of string type
		oriStr := value.ValueToString()
		tempBytes := []byte(oriStr)
		bytesL := len(tempBytes)
		oriByteL := int(commandInfo.Length * 2)
		if bytesL < oriByteL {
			less := make([]byte, oriByteL-bytesL)
			dataBytes = append(tempBytes, less...)
		} else if bytesL > oriByteL {
			dataBytes = tempBytes[:oriByteL]
		} else {
			dataBytes = []byte(oriStr)
		}
	}
	logs.Debug("Transfer CommandValue to dataBytes for write command, %v, %v", commandInfo.ValueType, dataBytes)
	return dataBytes, err
}

func calculateByteCount(commandInfo *CommandInfo) uint16 {
	var byteCount uint16
	if commandInfo.PrimaryTable == HOLDING_REGISTERS || commandInfo.PrimaryTable == INPUT_REGISTERS {
		byteCount = commandInfo.Length * 2
	} else {
		byteCount = commandInfo.Length
	}

	return byteCount
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
