package modbus

const (
	BOOL = "BOOL"

	INT16 = "INT16"
	INT32 = "INT32"
	INT64 = "INT64"

	UINT16 = "UINT16"
	UINT32 = "UINT32"
	UINT64 = "UINT64"

	FLOAT32 = "FLOAT32"
	FLOAT64 = "FLOAT64"

	DISCRETES_INPUT   = "DISCRETES_INPUT"
	COILS             = "COILS"
	INPUT_REGISTERS   = "INPUT_REGISTERS"
	HOLDING_REGISTERS = "HOLDING_REGISTERS"

	PRIMARY_TABLE    = "primaryTable"
	STARTING_ADDRESS = "startingAddress"
	IS_BYTE_SWAP     = "isByteSwap"
	IS_WORD_SWAP     = "isWordSwap"
	// RAW_TYPE define binary data type which read from Modbus device
	RAW_TYPE = "rawType"

	// STRING_REGISTER_SIZE  E.g. "abcd" need 4 bytes as is 2 registers(2 words), so STRING_REGISTER_SIZE=2
	STRING_REGISTER_SIZE   = "stringRegisterSize"
	SERVICE_STOP_WAIT_TIME = 1

	ValueTypeBool    = "Bool"
	ValueTypeString  = "String"
	ValueTypeUint8   = "Uint8"
	ValueTypeUint16  = "Uint16"
	ValueTypeUint32  = "Uint32"
	ValueTypeUint64  = "Uint64"
	ValueTypeInt8    = "Int8"
	ValueTypeInt16   = "Int16"
	ValueTypeInt32   = "Int32"
	ValueTypeInt64   = "Int64"
	ValueTypeFloat32 = "Float32"
	ValueTypeFloat64 = "Float64"
)

const (
	ProtocolTCP = "modbus-tcp"
	ProtocolRTU = "modbus-rtu"

	Address  = "Address"
	Port     = "Port"
	UnitID   = "UnitID"
	BaudRate = "BaudRate"
	DataBits = "DataBits"
	StopBits = "StopBits"
	// Parity: N - None, O - Odd, E - Even
	Parity = "Parity"

	Timeout     = "Timeout"
	IdleTimeout = "IdleTimeout"
)

var PrimaryTableBitCountMap = map[string]uint16{
	DISCRETES_INPUT:   1,
	COILS:             1,
	INPUT_REGISTERS:   16,
	HOLDING_REGISTERS: 16,
}

var ValueTypeBitCountMap = map[string]uint16{
	"Int16": 16,
	"Int32": 32,
	"Int64": 64,

	"Uint16": 16,
	"Uint32": 32,
	"Uint64": 64,

	"Float32": 32,
	"Float64": 64,

	"Bool":   1,
	"String": 16,
}
