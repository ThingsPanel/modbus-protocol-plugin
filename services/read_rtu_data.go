package services

import (
	"fmt"
	"io"
	"net"
	"time"
)

func ReadModbusRTUResponse(conn net.Conn) ([]byte, error) {
	// 设置读取超时
	err := conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	if err != nil {
		return nil, fmt.Errorf("设置读取超时失败: %v", err)
	}

	// 读取前3个字节以确定响应类型和长度
	header := make([]byte, 3)
	_, err = io.ReadFull(conn, header)
	if err != nil {
		return nil, fmt.Errorf("读取响应头失败: %v", err)
	}

	var length int
	if header[1]&0x80 != 0 {
		// 异常响应
		length = 5 // 从站地址(1) + 功能码(1) + 异常码(1) + CRC(2)
	} else {
		// 正常响应
		switch header[1] {
		case 0x01, 0x02, 0x03, 0x04:
			// 读取线圈状态、离散输入状态、保持寄存器或输入寄存器
			length = int(header[2]) + 5 // 从站地址(1) + 功能码(1) + 字节计数(1) + 数据(N) + CRC(2)
		case 0x05, 0x06:
			// 写入单个线圈或单个寄存器
			length = 8 // 从站地址(1) + 功能码(1) + 地址(2) + 值(2) + CRC(2)
		case 0x0F, 0x10:
			// 写入多个线圈或多个寄存器
			length = 8 // 从站地址(1) + 功能码(1) + 起始地址(2) + 数量(2) + CRC(2)
		default:
			return nil, fmt.Errorf("不支持的功能码: %02X", header[1])
		}
	}

	// 分配完整响应的缓冲区
	response := make([]byte, length)
	copy(response, header)

	// 读取剩余的字节
	_, err = io.ReadFull(conn, response[3:])
	if err != nil {
		return nil, fmt.Errorf("读取响应剩余部分失败: %v", err)
	}

	return response, nil
}
