package modbus

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"log"
)

type EndianessType string

const (
	BigEndian    EndianessType = "BIG"
	LittleEndian EndianessType = "LITTLE"
)

type MasterCommand struct {
	SlaveAddress    byte          // 从站地址
	FunctionCode    byte          // 功能码
	StartingAddress uint16        // 起始地址
	Quantity        uint16        // 寄存器数量或数据数量
	ValueData       []byte        // 写入的数据
	Endianess       EndianessType // 大端或小端
	Data            []byte        // 序列化后的数据
}

func NewCommand(requestType string, slaveAddress byte, functionCode byte, startingAddress uint16, quantity uint16, Endianess EndianessType) MasterCommand {
	return MasterCommand{
		SlaveAddress:    slaveAddress,
		FunctionCode:    functionCode,
		StartingAddress: startingAddress,
		Quantity:        quantity,
		Endianess:       Endianess, // 大端或小端
	}
}

func (c *MasterCommand) Serialize() ([]byte, error) {
	var buf bytes.Buffer

	// 写入从站地址
	buf.WriteByte(c.SlaveAddress)
	log.Println("----------", c.FunctionCode)
	// 根据功能码进行序列化
	switch c.FunctionCode {
	case 0x01, 0x02, 0x03, 0x04: // Read Coils, Read Discrete Inputs, Read Holding Registers, Read Input Registers
		buf.WriteByte(c.FunctionCode)
		if c.Endianess == BigEndian {
			binary.Write(&buf, binary.BigEndian, c.StartingAddress)
			binary.Write(&buf, binary.BigEndian, c.Quantity)
		} else {
			binary.Write(&buf, binary.LittleEndian, c.StartingAddress)
			binary.Write(&buf, binary.LittleEndian, c.Quantity)
		}

	case 0x05: // Write Single Coil ValueData为空返回错误
		buf.WriteByte(c.FunctionCode)
		binary.Write(&buf, binary.BigEndian, c.StartingAddress)
		if bytes.Equal(c.ValueData, []byte{0xFF, 0x00}) { // 假设ValueData字段用于存储线圈的值
			binary.Write(&buf, binary.BigEndian, uint16(0xFF00))
		} else if bytes.Equal(c.ValueData, []byte{0x00, 0x00}) {
			binary.Write(&buf, binary.BigEndian, uint16(0x0000))
		} else {
			return nil, fmt.Errorf("ValueData is not empty: %s", hex.EncodeToString(c.ValueData))
		}

	case 0x06: // Write Single Register ValueData为空返回错误
		if c.ValueData == nil {
			return nil, fmt.Errorf("ValueData is not empty")
		}
		buf.WriteByte(c.FunctionCode)

		if c.Endianess == BigEndian {
			binary.Write(&buf, binary.BigEndian, c.StartingAddress)
			binary.Write(&buf, binary.BigEndian, c.ValueData)
		} else {
			binary.Write(&buf, binary.LittleEndian, c.StartingAddress)
			binary.Write(&buf, binary.LittleEndian, c.ValueData)
		}

	// 我将这两个功能码合并，因为它们的序列化逻辑是相同的
	case 0x0F, 0x10: // Write Multiple Coils, Write Multiple Registers
		if c.ValueData == nil {
			return nil, fmt.Errorf("ValueData is not empty")
		}
		buf.WriteByte(c.FunctionCode)
		if c.Endianess == BigEndian {
			binary.Write(&buf, binary.BigEndian, c.StartingAddress)
			binary.Write(&buf, binary.BigEndian, c.ValueData)
		} else {
			binary.Write(&buf, binary.LittleEndian, c.StartingAddress)
			binary.Write(&buf, binary.LittleEndian, c.ValueData)
		}
		// 这里我们假设Data字段是[]byte类型并已经定义在Command结构中
		buf.WriteByte(byte(len(c.Data)))
		buf.Write(c.Data)

	default:
		return nil, fmt.Errorf("unsupported function code: %x", c.FunctionCode)
	}

	return buf.Bytes(), nil
}
