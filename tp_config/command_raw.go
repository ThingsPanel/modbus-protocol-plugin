package tpconfig

import (
	"encoding/binary"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/Knetic/govaluate"
	globaldata "github.com/ThingsPanel/modbus-protocol-plugin/global_data"
	"github.com/sirupsen/logrus"
)

type CommandRaw struct {
	FunctionCode    byte   // 功能码
	StartingAddress uint16 // 起始地址
	Quantity        uint16 // 寄存器数量或数据数量
	Endianess       string // 大端或小端 BIG or LITTLE

	Interval             int    // 采集时间间隔
	DataType             string // 数据类型 int16, uint16, int32, uint32, float32, float64
	DataIdetifierListStr string // 数据标识符 例如：A1, A2, A3...
	EquationListStr      string // 公式 例如：A1*0.1, A2*0.2, A3*0.3...
	DecimalPlacesListStr string // 小数位数 例如：1, 2, 3...
}

// NewCommandRaw 创建CommandRaw
func NewCommandRaw(commandRawMap map[string]interface{}) (*CommandRaw, error) {
	// 类型断言之前，先确保该key存在并且值的类型正确
	functionCode, ok := commandRawMap["FunctionCode"].(float64)
	if !ok {
		return nil, fmt.Errorf("functionCode is either missing or of incorrect type")
	}

	startingAddress, ok := commandRawMap["StartingAddress"].(float64)
	if !ok {
		return nil, fmt.Errorf("startingAddress is either missing or of incorrect type")
	}

	quantity, ok := commandRawMap["Quantity"].(float64)
	if !ok {
		return nil, fmt.Errorf("quantity is either missing or of incorrect type")
	}

	endianess, ok := commandRawMap["Endianess"].(string)
	if !ok {
		return nil, fmt.Errorf("endianess is either missing or of incorrect type")
	}

	interval, ok := commandRawMap["Interval"].(float64)
	if !ok {
		return nil, fmt.Errorf("interval is either missing or of incorrect type")
	}

	dataType, ok := commandRawMap["DataType"].(string)
	if !ok {
		return nil, fmt.Errorf("dataType is either missing or of incorrect type")
	}

	dataIdetifierListStr, ok := commandRawMap["DataIdentifierListStr"].(string)
	if !ok {
		return nil, fmt.Errorf("dataIdetifierListStr is either missing or of incorrect type")
	}

	equationListStr, ok := commandRawMap["EquationListStr"].(string)
	if !ok {
		equationListStr = ""
		logrus.Warn("equationListStr is either missing or of incorrect type, set to empty string")
	}

	decimalPlacesListStr, ok := commandRawMap["DecimalPlacesListStr"].(string)
	if !ok {
		decimalPlacesListStr = ""
		logrus.Warn("decimalPlacesListStr is either missing or of incorrect type, set to empty string")
	}
	// ... repeat the same for other fields ...

	return &CommandRaw{
		FunctionCode:         byte(functionCode),
		StartingAddress:      uint16(startingAddress),
		Quantity:             uint16(quantity),
		Endianess:            endianess,
		Interval:             int(interval),
		DataType:             dataType,
		DataIdetifierListStr: dataIdetifierListStr,
		EquationListStr:      equationListStr,
		DecimalPlacesListStr: decimalPlacesListStr,
	}, nil
}

// getNumericValue 检查值是否为数字类型，如果是则转换为 float64，否则返回错误
func getNumericValue(value interface{}) (float64, error) {
	if value == nil {
		return 0, fmt.Errorf("value is nil, expected numeric type")
	}

	switch v := value.(type) {
	case float64:
		return v, nil
	case float32:
		return float64(v), nil
	case int:
		return float64(v), nil
	case int32:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case uint:
		return float64(v), nil
	case uint32:
		return float64(v), nil
	case uint64:
		return float64(v), nil
	default:
		return 0, fmt.Errorf("value is not a numeric type, got %T: %v", value, value)
	}
}

// 根据CommandRaw计算写报文的功能码、起始地址、数据
func (c *CommandRaw) GetWriteCommand(key string, value interface{}, index int) (byte, uint16, []byte, error) {
	var functionCode byte
	var startingAddress uint16
	var data []byte

	// 根据c.StartingAddress、index和c.DataType
	// 计算出写报文的起始地址和数据
	switch c.DataType {
	case "int16":
		val, err := getNumericValue(value)
		if err != nil {
			return functionCode, startingAddress, data, err
		}
		startingAddress = c.StartingAddress + uint16(index)
		data = make([]byte, 2)
		if c.Endianess == "LITTLE" {
			binary.LittleEndian.PutUint16(data, uint16(val))
		} else {
			binary.BigEndian.PutUint16(data, uint16(val))
		}
		// 单寄存器数据使用功能码 0x06
		if c.FunctionCode == 0x03 || c.FunctionCode == 0x04 {
			functionCode = 0x06
		}
	case "uint16":
		val, err := getNumericValue(value)
		if err != nil {
			return functionCode, startingAddress, data, err
		}
		startingAddress = c.StartingAddress + uint16(index)
		data = make([]byte, 2)
		if c.Endianess == "LITTLE" {
			binary.LittleEndian.PutUint16(data, uint16(val))
		} else {
			binary.BigEndian.PutUint16(data, uint16(val))
		}
		// 单寄存器数据使用功能码 0x06
		if c.FunctionCode == 0x03 || c.FunctionCode == 0x04 {
			functionCode = 0x06
		}
	case "int32":
		val, err := getNumericValue(value)
		if err != nil {
			return functionCode, startingAddress, data, err
		}
		startingAddress = c.StartingAddress + uint16(index)*2
		data = make([]byte, 4)
		c.encodeUint32WithEndianess(data, uint32(int32(val)))
		// 多寄存器数据使用功能码 0x10
		if c.FunctionCode == 0x03 || c.FunctionCode == 0x04 {
			functionCode = 0x10
		}
	case "uint32":
		val, err := getNumericValue(value)
		if err != nil {
			return functionCode, startingAddress, data, err
		}
		startingAddress = c.StartingAddress + uint16(index)*2
		data = make([]byte, 4)
		c.encodeUint32WithEndianess(data, uint32(val))
		// 多寄存器数据使用功能码 0x10
		if c.FunctionCode == 0x03 || c.FunctionCode == 0x04 {
			functionCode = 0x10
		}
	case "int64":
		val, err := getNumericValue(value)
		if err != nil {
			return functionCode, startingAddress, data, err
		}
		startingAddress = c.StartingAddress + uint16(index)*4
		data = make([]byte, 8)
		c.encodeUint64WithEndianess(data, uint64(int64(val)))
		// 多寄存器数据使用功能码 0x10
		if c.FunctionCode == 0x03 || c.FunctionCode == 0x04 {
			functionCode = 0x10
		}
	case "float32":
		val, err := getNumericValue(value)
		if err != nil {
			return functionCode, startingAddress, data, err
		}
		startingAddress = c.StartingAddress + uint16(index)*2
		data = make([]byte, 4)
		bits := math.Float32bits(float32(val))
		c.encodeUint32WithEndianess(data, bits)
		// 多寄存器数据使用功能码 0x10
		if c.FunctionCode == 0x03 || c.FunctionCode == 0x04 {
			functionCode = 0x10
		}
	case "float64":
		val, err := getNumericValue(value)
		if err != nil {
			return functionCode, startingAddress, data, err
		}
		startingAddress = c.StartingAddress + uint16(index)*4
		data = make([]byte, 8)
		bits := math.Float64bits(val)
		c.encodeUint64WithEndianess(data, bits)
		// 多寄存器数据使用功能码 0x10
		if c.FunctionCode == 0x03 || c.FunctionCode == 0x04 {
			functionCode = 0x10
		}
	case "coil":
		startingAddress = c.StartingAddress + uint16(index)

		val, err := getNumericValue(value)
		if err != nil {
			return functionCode, startingAddress, data, err
		}

		if val == 1.0 {
			data = []byte{0xFF, 0x00}
		} else if val == 0.0 {
			data = []byte{0x00, 0x00}
		} else {
			// 返回无效值错误或其他错误处理逻辑
			return functionCode, startingAddress, data, fmt.Errorf("invalid coil value: %f", val)
		}
		// 线圈使用功能码 0x05
		if c.FunctionCode == 0x01 || c.FunctionCode == 0x02 {
			functionCode = 0x05
		}
	}
	return functionCode, startingAddress, data, nil

}

// 将modbus返回的数据序列化为json报文
func (c *CommandRaw) Serialize(resp []byte) (map[string]interface{}, error) {
	if len(resp) < 4 {
		return nil, fmt.Errorf("invalid response length: %d", len(resp))
	}
	// 检查Modbus异常响应
	if resp[1] >= byte(0x80) {
		// 错误码映射
		err := fmt.Errorf("function Code(0x%02x) exception Code(0x%02x):%s", resp[1], resp[2], globaldata.GetModbusErrorDesc(resp[2]))
		logrus.Error(err)
		return nil, err
	}

	data := resp[3:] // 过滤Modbus地址、功能码和字节计数
	values := make(map[string]interface{})

	// Choose the right byte order based on the Endianess attribute
	// 注意：BADC 和 CDAB 仅影响多寄存器数据（int32/uint32/float32/int64/float64）
	// 对于单寄存器数据（int16/uint16），BADC 和 CDAB 使用大端序
	var byteOrder binary.ByteOrder
	if c.Endianess == "LITTLE" {
		byteOrder = binary.LittleEndian
	} else if c.Endianess == "BIG" || c.Endianess == "BADC" || c.Endianess == "CDAB" {
		// BADC 和 CDAB 对单寄存器不生效，使用大端序
		byteOrder = binary.BigEndian
	} else {
		return nil, fmt.Errorf("unknown endianess specified: %s", c.Endianess)
	}

	dataIds := strings.Split(c.DataIdetifierListStr, ",")
	byteIndex := 0
	for _, id := range dataIds {
		var val float64
		id = strings.TrimSpace(id)
		switch c.DataType {
		case "int16":
			val = float64(int16(byteOrder.Uint16(data[byteIndex : byteIndex+2])))
			byteIndex += 2
		case "uint16":
			val = float64(byteOrder.Uint16(data[byteIndex : byteIndex+2]))
			byteIndex += 2
		case "int32":
			val = float64(int32(c.parseUint32WithEndianess(data[byteIndex : byteIndex+4])))
			byteIndex += 4
		case "uint32":
			val = float64(c.parseUint32WithEndianess(data[byteIndex : byteIndex+4]))
			byteIndex += 4
		case "float32":
			bits := c.parseUint32WithEndianess(data[byteIndex : byteIndex+4])
			val = float64(math.Float32frombits(bits))
			byteIndex += 4
		case "float64":
			val = math.Float64frombits(c.parseUint64WithEndianess(data[byteIndex : byteIndex+8]))
			byteIndex += 8
		case "coil":
			// Assuming each coil is a single bit and we may need to read multiple coils
			// stored in consecutive bits of bytes.
			coilVal := (data[byteIndex/8] >> (byteIndex % 8)) & 0x01 // Extract the bit at the correct position
			val = float64(coilVal)
			byteIndex++ // Move to the next bit
		}

		values[id] = val
	}

	// 以下是公式处理和小数处理

	decimalPlacesList := strings.Split(c.DecimalPlacesListStr, ",")
	equations := strings.Split(c.EquationListStr, ",")

	singleDecimalPlace := false
	singleEquation := false

	// 如果只有一个小数位，将其应用于所有数据
	if len(decimalPlacesList) == 1 {
		singleDecimalPlace = true
	}

	// 如果只有一个公式，将其应用于所有数据
	if len(equations) == 1 {
		singleEquation = true
	}

	for i, id := range dataIds {
		id = strings.TrimSpace(id)

		// 处理公式
		if c.EquationListStr != "" && (i < len(equations) || singleEquation) {
			eqIndex := i
			if singleEquation {
				eqIndex = 0
			}

			expression, err := govaluate.NewEvaluableExpression(equations[eqIndex])
			if err != nil {
				return nil, err
			}

			result, err := expression.Evaluate(values)
			if err != nil {
				return nil, err
			}

			resFloat, ok := result.(float64)
			if !ok {
				return nil, fmt.Errorf("result of equation is not float64")
			}
			values[id] = resFloat
		}

		// 处理小数位数
		if c.DecimalPlacesListStr != "" && (i < len(decimalPlacesList) || singleDecimalPlace) {
			placeIndex := i
			if singleDecimalPlace {
				placeIndex = 0
			}

			places, err := strconv.Atoi(strings.TrimSpace(decimalPlacesList[placeIndex]))
			if err != nil {
				logrus.Info("invalid decimal place value for ", id, ": ", err)
				continue
			}

			multiplier := math.Pow(10, float64(places))
			if val, ok := values[id].(float64); ok {
				values[id] = math.Round(val*multiplier) / multiplier
			} else {
				logrus.Info("value of ", id, " is not float64")
				continue
			}
		}
	}

	return values, nil
}

// parseUint32WithEndianess 根据字节序解析 4 字节数据（32位）
func (c *CommandRaw) parseUint32WithEndianess(data []byte) uint32 {
	switch c.Endianess {
	case "BIG": // ABCD
		return binary.BigEndian.Uint32(data)
	case "LITTLE": // DCBA
		return binary.LittleEndian.Uint32(data)
	case "BADC": // Byte Swap - 每个寄存器内字节交换
		// [Byte2 Byte1][Byte4 Byte3]
		return uint32(data[1])<<24 | uint32(data[0])<<16 | uint32(data[3])<<8 | uint32(data[2])
	case "CDAB": // Word Swap + Byte Swap - 寄存器交换
		// [Byte3 Byte4][Byte1 Byte2]
		return uint32(data[2])<<24 | uint32(data[3])<<16 | uint32(data[0])<<8 | uint32(data[1])
	default:
		// 默认使用大端
		return binary.BigEndian.Uint32(data)
	}
}

// parseUint64WithEndianess 根据字节序解析 8 字节数据（64位）
func (c *CommandRaw) parseUint64WithEndianess(data []byte) uint64 {
	switch c.Endianess {
	case "BIG": // ABCDEFGH
		return binary.BigEndian.Uint64(data)
	case "LITTLE": // HGFEDCBA
		return binary.LittleEndian.Uint64(data)
	case "BADC": // Byte Swap - 每个寄存器内字节交换
		// [Byte2 Byte1][Byte4 Byte3][Byte6 Byte5][Byte8 Byte7]
		return uint64(data[1])<<56 | uint64(data[0])<<48 | uint64(data[3])<<40 | uint64(data[2])<<32 |
			uint64(data[5])<<24 | uint64(data[4])<<16 | uint64(data[7])<<8 | uint64(data[6])
	case "CDAB": // Word Swap + Byte Swap - 寄存器交换
		// [Byte3 Byte4][Byte1 Byte2][Byte7 Byte8][Byte5 Byte6]
		return uint64(data[2])<<56 | uint64(data[3])<<48 | uint64(data[0])<<40 | uint64(data[1])<<32 |
			uint64(data[6])<<24 | uint64(data[7])<<16 | uint64(data[4])<<8 | uint64(data[5])
	default:
		// 默认使用大端
		return binary.BigEndian.Uint64(data)
	}
}

// encodeUint32WithEndianess 根据字节序编码 4 字节数据（32位）
func (c *CommandRaw) encodeUint32WithEndianess(data []byte, value uint32) {
	switch c.Endianess {
	case "BIG": // ABCD
		binary.BigEndian.PutUint32(data, value)
	case "LITTLE": // DCBA
		binary.LittleEndian.PutUint32(data, value)
	case "BADC": // Byte Swap - 每个寄存器内字节交换
		// [Byte2 Byte1][Byte4 Byte3]
		data[0] = byte(value >> 16)
		data[1] = byte(value >> 24)
		data[2] = byte(value)
		data[3] = byte(value >> 8)
	case "CDAB": // Word Swap + Byte Swap - 寄存器交换
		// [Byte3 Byte4][Byte1 Byte2]
		data[0] = byte(value >> 8)
		data[1] = byte(value)
		data[2] = byte(value >> 24)
		data[3] = byte(value >> 16)
	default:
		// 默认使用大端
		binary.BigEndian.PutUint32(data, value)
	}
}

// encodeUint64WithEndianess 根据字节序编码 8 字节数据（64位）
func (c *CommandRaw) encodeUint64WithEndianess(data []byte, value uint64) {
	switch c.Endianess {
	case "BIG": // ABCDEFGH
		binary.BigEndian.PutUint64(data, value)
	case "LITTLE": // HGFEDCBA
		binary.LittleEndian.PutUint64(data, value)
	case "BADC": // Byte Swap - 每个寄存器内字节交换
		// [Byte2 Byte1][Byte4 Byte3][Byte6 Byte5][Byte8 Byte7]
		data[0] = byte(value >> 48)
		data[1] = byte(value >> 56)
		data[2] = byte(value >> 32)
		data[3] = byte(value >> 40)
		data[4] = byte(value >> 16)
		data[5] = byte(value >> 24)
		data[6] = byte(value)
		data[7] = byte(value >> 8)
	case "CDAB": // Word Swap + Byte Swap - 寄存器交换
		// [Byte3 Byte4][Byte1 Byte2][Byte7 Byte8][Byte5 Byte6]
		data[0] = byte(value >> 40)
		data[1] = byte(value >> 32)
		data[2] = byte(value >> 56)
		data[3] = byte(value >> 48)
		data[4] = byte(value >> 8)
		data[5] = byte(value)
		data[6] = byte(value >> 24)
		data[7] = byte(value >> 16)
	default:
		// 默认使用大端
		binary.BigEndian.PutUint64(data, value)
	}
}
