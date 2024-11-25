package mqtt

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"

	"github.com/sirupsen/logrus"
)

func ReadModbusRTUResponse(conn net.Conn, regPkg string) ([]byte, error) {
	var heartbeatResponse []byte

	// 第一个字节预读
	firstByte := make([]byte, 1)
	_, err := io.ReadFull(conn, firstByte)
	if err != nil {
		logrus.Warn("读取第一个字节失败:", err)
		return nil, fmt.Errorf("read failed")
	}

	// 判断是否可能是心跳包
	if regPkg != "" && firstByte[0] == regPkg[0] {
		// 可能是心跳包，继续读取剩余部分
		expectedBytes, _ := hex.DecodeString(regPkg)
		if len(expectedBytes) > 1 {
			remainingBytes := make([]byte, len(expectedBytes)-1)
			_, err = io.ReadFull(conn, remainingBytes)
			if err != nil {
				logrus.Warn("读取心跳包剩余数据失败:", err)
				return nil, fmt.Errorf("read failed")
			}

			heartbeatResponse = append(firstByte, remainingBytes...)
			if bytes.Equal(heartbeatResponse, expectedBytes) {
				logrus.Debug("成功读取到心跳包响应")
			}
		}

		// 继续读取 Modbus RTU 响应的第一个字节
		_, err = io.ReadFull(conn, firstByte)
		if err != nil {
			if heartbeatResponse != nil {
				return heartbeatResponse, nil
			}
			logrus.Warn("读取失败:", err)
			return nil, fmt.Errorf("read failed")
		}
	}

	// 继续读取剩余的2个字节以确定响应类型和长度
	remainingHeader := make([]byte, 2)
	_, err = io.ReadFull(conn, remainingHeader)
	if err != nil {
		if heartbeatResponse != nil {
			return heartbeatResponse, nil
		}
		logrus.Warn("读取失败:", err)
		return nil, fmt.Errorf("read failed")
	}

	// 组合header
	header := append(firstByte, remainingHeader...)

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
			logrus.Warnf("不支持的功能码: %02X，已丢弃所有剩余数据", header[1])
			return nil, errors.New("not supported function code")
		}
	}

	// 分配完整响应的缓冲区
	response := make([]byte, length)
	copy(response, header)

	// 读取剩余的字节
	_, err = io.ReadFull(conn, response[3:])
	if err != nil {
		if heartbeatResponse != nil {
			return heartbeatResponse, nil
		}
		logrus.Warn("读取剩余数据失败,跳过:", err)
		return nil, fmt.Errorf("read failed")
	}

	// 如果有心跳包响应，将其与 Modbus 响应组合
	if heartbeatResponse != nil {
		finalResponse := append(heartbeatResponse, response...)
		return finalResponse, nil
	}

	return response, nil
}
