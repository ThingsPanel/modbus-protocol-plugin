package services

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"

	"github.com/sirupsen/logrus"
)

func ReadModbusTCPResponse(conn net.Conn) ([]byte, error) {
	// 读取 MBAP 头 (7 bytes) + 功能码 (1 byte)
	header := make([]byte, 8)
	_, err := io.ReadFull(conn, header)
	if err != nil {
		logrus.Warn("读取 MBAP 头失败:", err)
		return nil, fmt.Errorf("read MBAP header failed")
	}

	// 解析 MBAP 头
	transactionID := binary.BigEndian.Uint16(header[0:2])
	protocolID := binary.BigEndian.Uint16(header[2:4])
	length := binary.BigEndian.Uint16(header[4:6])
	unitID := header[6]
	functionCode := header[7]

	// 检查协议 ID
	if protocolID != 0 {
		return nil, errors.New("invalid Modbus TCP protocol ID")
	}

	// 计算剩余数据长度（长度字段包括单元 ID 和之后的所有数据）
	remainingLength := int(length) - 2 // 减去单元 ID 和功能码的长度

	var responseLength int
	if functionCode&0x80 != 0 {
		// 异常响应
		responseLength = 1 // 异常码
	} else {
		// 正常响应
		switch functionCode {
		case 0x01, 0x02, 0x03, 0x04:
			// 读取线圈状态、离散输入状态、保持寄存器或输入寄存器
			responseLength = remainingLength // 字节计数(1) + 数据(N)
		case 0x05, 0x06:
			// 写入单个线圈或单个寄存器
			responseLength = 4 // 输出地址(2) + 输出值(2)
		case 0x0F, 0x10:
			// 写入多个线圈或多个寄存器
			responseLength = 4 // 起始地址(2) + 数量(2)
		default:
			logrus.Warnf("不支持的功能码: %02X", functionCode)
			return nil, errors.New("unsupported function code")
		}
	}

	// 分配完整响应的缓冲区
	response := make([]byte, 8+responseLength)
	copy(response, header)

	// 读取剩余的字节
	_, err = io.ReadFull(conn, response[8:])
	if err != nil {
		logrus.Warn("读取剩余数据失败:", err)
		return nil, fmt.Errorf("read remaining data failed")
	}

	logrus.Debugf("Received Modbus TCP response: TransactionID=%d, UnitID=%d, FunctionCode=%02X, Length=%d",
		transactionID, unitID, functionCode, len(response))

	return response, nil
}
