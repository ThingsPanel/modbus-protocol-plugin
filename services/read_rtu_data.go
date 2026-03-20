package services

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"net"

	"github.com/sirupsen/logrus"
)

// ReadModbusRTUResponse 读取Modbus RTU响应
// expectedFuncCode: 当前请求的功能码，用于验证响应
func ReadModbusRTUResponse(conn net.Conn, expectedFuncCode byte) ([]byte, error) {
	var buffer bytes.Buffer
	readBuffer := make([]byte, 256)

	// 最多读取两次
	for i := 0; i < 2; i++ {
		n, err := conn.Read(readBuffer)
		logrus.Info("---------读取数据详情(第", i+1, "次)---------")
		logrus.Info("读取字节数: ", n)
		if n > 0 {
			logrus.Info("数据内容(hex): ", hex.EncodeToString(readBuffer[:n]))
			logrus.Info("数据内容(bytes): ", readBuffer[:n])
		}
		if err != nil {
			logrus.Info("读取错误: ", err)
			logrus.Info("-----------------------------")
			// 超时错误不算连接异常，允许继续处理
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				break
			}
			return nil, fmt.Errorf("连接异常: %v", err)
		}
		logrus.Info("-----------------------------")

		buffer.Write(readBuffer[:n])

		// 尝试解析modbus响应，必须匹配功能码
		if modbusData := findModbusResponse(buffer.Bytes(), expectedFuncCode); modbusData != nil {
			return modbusData, nil
		}
	}

	// 超时但没找到响应，返回nil
	return nil, nil
}

// findModbusResponse 在数据中查找有效的Modbus响应
// 必须匹配 expectedFuncCode，否则返回nil
func findModbusResponse(data []byte, expectedFuncCode byte) []byte {
	if len(data) < 5 {
		return nil
	}

	for i := 0; i < len(data)-4; i++ {
		// 跳过0x30-0x39（数字字符干扰）
		if data[i] >= 0x30 && data[i] <= 0x39 {
			continue
		}

		// 检查功能码是否匹配
		funcCode := data[i+1]

		// 如果不是异常响应，必须匹配请求的功能码
		if funcCode&0x80 == 0 && funcCode != expectedFuncCode {
			continue
		}

		// 检查功能码是否有效
		if !isValidFunctionCode(funcCode) {
			continue
		}

		respLen, err := calculateResponseLength(data[i:])
		if err != nil {
			continue
		}

		if i+respLen <= len(data) {
			return data[i : i+respLen]
		}
	}
	return nil
}

func isValidFunctionCode(code byte) bool {
	// 异常响应（最高位为1）也是有效的
	if code&0x80 != 0 {
		return true
	}
	// 检查正常功能码
	validCodes := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x0F, 0x10}
	for _, valid := range validCodes {
		if code == valid {
			return true
		}
	}
	return false
}

func calculateResponseLength(header []byte) (int, error) {
	if header[1]&0x80 != 0 {
		return 5, nil // 异常响应固定5字节
	}

	switch header[1] {
	case 0x01, 0x02:
		return int(header[2]) + 5, nil
	case 0x03, 0x04:
		return int(header[2]) + 5, nil
	case 0x05, 0x06, 0x0F, 0x10:
		return 8, nil
	default:
		return 0, fmt.Errorf("不支持的功能码: %02X", header[1])
	}
}
