package tpconfig

import (
	"encoding/binary"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/Knetic/govaluate"
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
		return nil, fmt.Errorf("equationListStr is either missing or of incorrect type")
	}

	decimalPlacesListStr, ok := commandRawMap["DecimalPlacesListStr"].(string)
	if !ok {
		return nil, fmt.Errorf("decimalPlacesListStr is either missing or of incorrect type")
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

// 根据CommandRaw计算写报文的功能码、起始地址、数据
func (c *CommandRaw) GetWriteCommand(key string, value interface{}, index int) (byte, uint16, []byte, error) {
	var functionCode byte
	var startingAddress uint16
	var data []byte
	// 找到
	switch c.FunctionCode {
	case 0x01:
		functionCode = 0x05
	case 0x02:
		functionCode = 0x05
	case 0x03:
		functionCode = 0x06
	case 0x04:
		functionCode = 0x06
	}
	// 根据c.StartingAddress、index和c.DataType
	// 计算出写报文的起始地址和数据
	switch c.DataType {
	case "int16":
		startingAddress = c.StartingAddress + uint16(index)*2
		data = make([]byte, 2)
		if c.Endianess == "LITTLE" {
			binary.LittleEndian.PutUint16(data, uint16(value.(int16)))
		} else {
			binary.BigEndian.PutUint16(data, uint16(value.(int16)))
		}
	case "uint16":
		startingAddress = c.StartingAddress + uint16(index)*2
		data = make([]byte, 2)
		if c.Endianess == "LITTLE" {
			binary.LittleEndian.PutUint16(data, value.(uint16))
		} else {
			binary.BigEndian.PutUint16(data, value.(uint16))
		}
	case "int32":
		startingAddress = c.StartingAddress + uint16(index)*4
		data = make([]byte, 4)
		if c.Endianess == "LITTLE" {
			binary.LittleEndian.PutUint32(data, uint32(value.(int32)))
		} else {
			binary.BigEndian.PutUint32(data, uint32(value.(int32)))
		}
	case "uint32":
		startingAddress = c.StartingAddress + uint16(index)*4
		data = make([]byte, 4)
		if c.Endianess == "LITTLE" {
			binary.LittleEndian.PutUint32(data, value.(uint32))
		} else {
			binary.BigEndian.PutUint32(data, value.(uint32))
		}
	case "float32":
		startingAddress = c.StartingAddress + uint16(index)*4
		data = make([]byte, 4)
		if c.Endianess == "LITTLE" {
			bits := math.Float32bits(float32(value.(float64)))
			binary.LittleEndian.PutUint32(data, bits)
		} else {
			bits := math.Float32bits(float32(value.(float64)))
			binary.BigEndian.PutUint32(data, bits)
		}
	case "float64":
		startingAddress = c.StartingAddress + uint16(index)*8
		data = make([]byte, 8)
		if c.Endianess == "LITTLE" {
			bits := math.Float64bits(value.(float64))
			binary.LittleEndian.PutUint64(data, bits)
		} else {
			bits := math.Float64bits(value.(float64))
			binary.BigEndian.PutUint64(data, bits)
		}
	case "coil":
		startingAddress = c.StartingAddress + uint16(index)

		val, ok := value.(float64)
		if !ok {
			// 返回类型错误或其他错误处理逻辑
			return functionCode, startingAddress, data, fmt.Errorf("expected float64 value for coil, got %T", value)
		}

		if val == 1.0 {
			data = []byte{0xFF, 0x00}
		} else if val == 0.0 {
			data = []byte{0x00, 0x00}
		} else {
			// 返回无效值错误或其他错误处理逻辑
			return functionCode, startingAddress, data, fmt.Errorf("invalid coil value: %f", val)
		}
	}
	return functionCode, startingAddress, data, nil

}

// 将modbus返回的数据序列化为json报文
func (c *CommandRaw) Serialize(resp []byte) (map[string]interface{}, error) {
	// 检查Modbus异常响应
	if resp[1]&0x80 != 0 {
		return nil, fmt.Errorf("modbus exception response: exception code %d", resp[2])
	}

	data := resp[3:] // 过滤Modbus地址、功能码和字节计数
	values := make(map[string]interface{})

	// Choose the right byte order based on the Endianess attribute
	var byteOrder binary.ByteOrder
	if c.Endianess == "LITTLE" {
		byteOrder = binary.LittleEndian
	} else if c.Endianess == "BIG" {
		byteOrder = binary.BigEndian
	} else {
		return nil, fmt.Errorf("unknown endianess specified")
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
			val = float64(int32(byteOrder.Uint32(data[byteIndex : byteIndex+4])))
			byteIndex += 4
		case "uint32":
			val = float64(byteOrder.Uint32(data[byteIndex : byteIndex+4]))
			byteIndex += 4
		case "float32":
			var floatVal float32
			bits := byteOrder.Uint32(data[byteIndex : byteIndex+4])
			floatVal = math.Float32frombits(bits)
			val = float64(floatVal)
			byteIndex += 4
		case "float64":
			val = math.Float64frombits(byteOrder.Uint64(data[byteIndex : byteIndex+8]))
			byteIndex += 8
		case "coil":
			val = float64(data[byteIndex])
			byteIndex++
			// ... 处理其他数据类型
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
