package modbus

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/sirupsen/logrus"
)

var globalTransactionID uint32

type TCPCommand struct {
	RequestTransactionID uint16 // 请求时使用的TransactionID
	ProtocolID           uint16 // 协议标识符, typically 0 for Modbus
	Length               uint16 // 后续的字节数，包括单元标识符和数据
	MasterCommand        // 嵌入Command结构
}

func NewTCPCommand(slaveAddress byte, functionCode byte, startingAddress uint16, quantity uint16, endianess EndianessType) TCPCommand {
	// 使用原子操作获取自增的TransactionID
	id := atomic.AddUint32(&globalTransactionID, 1)
	return TCPCommand{
		RequestTransactionID: uint16(id),
		ProtocolID:           0, // Typically, this is 0 for Modbus
		MasterCommand: MasterCommand{
			SlaveAddress:    slaveAddress,
			FunctionCode:    functionCode,
			StartingAddress: startingAddress,
			Quantity:        quantity,
			Endianess:       endianess,
		},
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
	binary.Write(&buf, binary.BigEndian, t.RequestTransactionID)
	binary.Write(&buf, binary.BigEndian, t.ProtocolID)
	binary.Write(&buf, binary.BigEndian, t.Length)
	buf.Write(data)

	return buf.Bytes(), nil
}

const MBAPHeaderLength = 7 // MBAP Header length for Modbus TCP (TransactionID:2 + ProtocolID:2 + Length:2 + UnitID:1)

// 解析Modbus TCP返回的数据并提取数据部分
func (t *TCPCommand) ParseTCPResponse(resp []byte) ([]byte, error) {
	if len(resp) < MBAPHeaderLength {
		logrus.Error("response too short")
		return nil, errors.New("response too short")
	}

	// 提取MBAP字段
	respTransactionID := binary.BigEndian.Uint16(resp[0:2])
	respProtocolID := binary.BigEndian.Uint16(resp[2:4])
	respLength := binary.BigEndian.Uint16(resp[4:6])

	// 校验TransactionID是否匹配
	if respTransactionID != t.RequestTransactionID {
		logrus.Warnf("TransactionID mismatch: sent=%d, received=%d", t.RequestTransactionID, respTransactionID)
		return nil, fmt.Errorf("TransactionID mismatch: sent=%d, received=%d", t.RequestTransactionID, respTransactionID)
	}

	// Check if the ProtocolID is as expected (typically 0 for Modbus)
	if respProtocolID != 0 {
		logrus.Error("unexpected ProtocolID")
		return nil, fmt.Errorf("unexpected ProtocolID: %d", respProtocolID)
	}

	// 校验长度
	expectedLen := int(respLength) + 6
	if len(resp) != expectedLen {
		logrus.Error("length mismatch")
		return nil, fmt.Errorf("length mismatch: MBAP header reports %d bytes but received %d bytes", respLength, len(resp)-6)
	}

	// Extract and return the PDU (Protocol Data Unit) without the MBAP header
	return resp[MBAPHeaderLength:], nil
}
