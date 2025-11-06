package services

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"fmt"
	"net"

	"github.com/sirupsen/logrus"
)

/*
逻辑：
一次请求必然有一次响应。
在发送请求前，也许缓冲区已经有了多个心跳包，也许没有心跳包。
程序需要跳过心跳包，来读取响应数据。
在一次读取的时候也许响应会跟在心跳包后。
只要有响应，就返回响应。
ReadModbusRTUResponse方法返回失败会导致连接关闭，所以如果不是连接断开就不能返回错误。
不做CRC校验
*/

func isTimeout(err error) bool {
	if netErr, ok := err.(net.Error); ok {
		return netErr.Timeout()
	}
	return false
}

func ReadModbusRTUResponse(conn net.Conn, regPkg string) ([]byte, error) {
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
			if !isTimeout(err) {
				return nil, fmt.Errorf("连接异常: %v", err)
			}
			break
		}
		logrus.Info("-----------------------------")

		buffer.Write(readBuffer[:n])

		// 尝试解析modbus响应
		if modbusData := findModbusResponse(buffer.Bytes()); modbusData != nil {
			return modbusData, nil
		}

		// 第一次没找到响应且没超时,继续读取
		if i == 0 && err == nil {
			continue
		}
	}

	return nil, nil
}

func findModbusResponse(data []byte) []byte {
	if len(data) < 5 {
		return nil
	}

	for i := 0; i < len(data)-4; i++ {
		if data[i] >= 0x30 && data[i] <= 0x39 {
			continue
		}

		if !isValidFunctionCode(data[i+1]) {
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
	// 检查是否为异常响应（功能码最高位为1）
	if code&0x80 != 0 {
		return true // 异常响应也是有效的
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
func ReadHeader(reader *bufio.Reader) ([]byte, error) {
	header, err := reader.Peek(3)
	if err != nil {
		return nil, err
	}
	_, err = reader.Discard(3)
	return header, err
}
