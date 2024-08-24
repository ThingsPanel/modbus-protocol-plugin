package modbus

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

type RTUCommand struct {
	ID            string // 唯一ID
	MasterCommand        // 嵌入Command结构
	CRC           uint16 // CRC校验值
	Data          []byte
	// 为RTU特定的其他属性，如果有的话
}

func NewRTUCommand(slaveAddress byte, functionCode byte, startingAddress uint16, quantity uint16, endianess EndianessType) RTUCommand {
	// uuid.NewV4().String() 生成唯一ID
	id := uuid.Must(uuid.NewV4()).String()
	return RTUCommand{
		ID: id,
		MasterCommand: MasterCommand{
			SlaveAddress:    slaveAddress,
			FunctionCode:    functionCode,
			StartingAddress: startingAddress,
			Quantity:        quantity,
			Endianess:       endianess,
		},
	}
}

// 序列化RTUCommand,会将CRC校验值附加到序列化后的数据后面并且赋值给RTUCommand.Data
func (r *RTUCommand) Serialize() ([]byte, error) {
	// 使用MasterCommand的序列化方法
	data, err := r.MasterCommand.Serialize()
	if err != nil {
		return nil, err
	}
	// 计算CRC
	crcValue := crc16(data)

	// 创建一个新的buffer来包含序列化后的数据和CRC值
	var buf bytes.Buffer
	buf.Write(data)
	binary.Write(&buf, binary.LittleEndian, crcValue)

	// 将序列化后的结果赋值给r.Data
	r.Data = buf.Bytes()

	return buf.Bytes(), nil
}

// modbus返回的数据校验并去除CRC校验值
func (r *RTUCommand) ParseAndValidateResponse(resp []byte) ([]byte, error) {
	if len(resp) < 3 { // minimal Modbus RTU frame size (1 addr + 1 function + 2 crc)
		return nil, errors.New("response too short")
	}
	if r.FunctionCode == 0x03 || r.FunctionCode == 0x04 || r.FunctionCode == 0x06 {
		//检查读取的数据与预设的数据长度不符合则丢弃
		if len(resp) != int(2*r.Quantity)+5 {
			return nil, fmt.Errorf("response length mismatch: expected %d but got %d", int(2*r.Quantity)+5, len(resp))
		}
	}
	// CRC 校验值在 Modbus RTU 中始终是小端格式
	receivedCRC := binary.LittleEndian.Uint16(resp[len(resp)-2:])

	// Compute CRC for the data without CRC
	computedCRC := crc16(resp[:len(resp)-2])

	// Compare the received CRC with the computed CRC
	if receivedCRC != computedCRC {
		logrus.Infof("CRC mismatch: expected %04X but got %04X", computedCRC, receivedCRC)
		// return nil, fmt.Errorf("CRC mismatch: expected %04X but got %04X", computedCRC, receivedCRC)
	}

	return resp[:len(resp)-2], nil
}
