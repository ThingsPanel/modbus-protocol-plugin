package modbus

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/gofrs/uuid"
)

type TCPCommand struct {
	ID            string // 唯一ID
	MasterCommand        // 嵌入Command结构
	Data          []byte
	TransactionID uint16 // 事务标识符
	ProtocolID    uint16 // 协议标识符, typically 0 for Modbus
	Length        uint16 // 后续的字节数，包括单元标识符和数据
}

func NewTCPCommand(slaveAddress byte, functionCode byte, startingAddress uint16, quantity uint16, endianess EndianessType) TCPCommand {
	// uuid.NewV4().String() 生成唯一ID
	id := uuid.Must(uuid.NewV4()).String()
	return TCPCommand{
		ID: id,
		MasterCommand: MasterCommand{
			SlaveAddress:    slaveAddress,
			FunctionCode:    functionCode,
			StartingAddress: startingAddress,
			Quantity:        quantity,
			Endianess:       endianess,
		},
		ProtocolID: 0, // Typically, this is 0 for Modbus
	}
}

// 序列化TCPCommand
func (t *TCPCommand) Serialize() ([]byte, error) {
	// 使用MasterCommand的序列化方法
	data, err := t.MasterCommand.Serialize()
	if err != nil {
		return nil, err
	}

	// 设置长度字段，长度字段是后续的字节数，包括单元标识符和数据
	t.Length = uint16(len(data))

	// 创建一个新的buffer来包含TCP Modbus头和数据
	var buf bytes.Buffer
	binary.Write(&buf, binary.BigEndian, t.TransactionID)
	binary.Write(&buf, binary.BigEndian, t.ProtocolID)
	binary.Write(&buf, binary.BigEndian, t.Length)
	// buf.WriteByte(t.SlaveAddress) // 写入单元标识符
	buf.Write(data)

	// 将序列化后的结果赋值给t.Data
	t.Data = buf.Bytes()

	return buf.Bytes(), nil
}

const MBAPHeaderLength = 6 // MBAP Header length for Modbus TCP

// 解析Modbus TCP返回的数据并提取数据部分
func (r *TCPCommand) ParseTCPResponse(resp []byte) ([]byte, error) {
	if len(resp) < MBAPHeaderLength { // minimal Modbus TCP frame size
		return nil, errors.New("response too short")
	}

	// Extracting MBAP fields from the response
	r.TransactionID = binary.BigEndian.Uint16(resp[0:2])
	r.ProtocolID = binary.BigEndian.Uint16(resp[2:4])
	r.Length = binary.BigEndian.Uint16(resp[4:6])

	// Check if the ProtocolID is as expected (typically 0 for Modbus)
	if r.ProtocolID != 0 {
		return nil, fmt.Errorf("unexpected ProtocolID: %d", r.ProtocolID)
	}

	// Check if the length field in the MBAP header matches the actual response length
	if int(r.Length)+6 != len(resp) { // +6 because we don't include the 6 byte length of the MBAP header and 1 byte unit identifier in the length field of the MBAP header
		return nil, fmt.Errorf("length mismatch: MBAP header reports %d bytes but received %d bytes", r.Length, len(resp)-MBAPHeaderLength)
	}

	// Extract and return the PDU (Protocol Data Unit) without the MBAP header
	return resp[MBAPHeaderLength:], nil
}
