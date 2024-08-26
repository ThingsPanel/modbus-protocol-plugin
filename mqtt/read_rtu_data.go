package mqtt

import (
	"errors"
	"fmt"
	"io"
	"net"

	"github.com/sirupsen/logrus"
)

func ReadModbusRTUResponse(conn net.Conn) ([]byte, error) {

	// 读取前3个字节以确定响应类型和长度
	header := make([]byte, 3)
	_, err := io.ReadFull(conn, header)
	if err != nil {
		logrus.Warn("读取失败,跳过:", err)
		return nil, fmt.Errorf("read failed")
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
			// 不支持的功能码，读取并丢弃所有剩余数据
			discardBuffer := make([]byte, 1024)
			for {
				_, err := conn.Read(discardBuffer)
				if err != nil {
					if err == io.EOF {
						break
					}
					logrus.Warn("读取不支持的功能码剩余数据失败,跳过:", err)
					return nil, fmt.Errorf("read failed")
				}
			}
			logrus.Warnf("不支持的功能码: %02X，已丢弃所有剩余数据%02X", header[1], discardBuffer)
			return nil, errors.New("not supported function code")
		}
	}

	// 分配完整响应的缓冲区
	response := make([]byte, length)
	copy(response, header)

	// 读取剩余的字节
	_, err = io.ReadFull(conn, response[3:])
	if err != nil {
		logrus.Warn("读取剩余数据失败,跳过:", err)
		return nil, fmt.Errorf("read failed")
	}

	return response, nil
}
