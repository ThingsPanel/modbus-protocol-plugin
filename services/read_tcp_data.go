package services

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"net"

	"github.com/sirupsen/logrus"
)

func ReadModbusTCPResponse(conn net.Conn, regPkg string) ([]byte, error) {
	var heartbeatResponse []byte

	// 第一个字节预读
	firstByte := make([]byte, 1)
	_, err := io.ReadFull(conn, firstByte)
	if err != nil {
		logrus.Warn("读取第一个字节失败:", err)
		return nil, fmt.Errorf("读取数据失败")
	}

	// 快速判断是否可能是心跳包（通过第一个字节特征）
	if regPkg != "" && firstByte[0] == regPkg[0] {
		// 可能是心跳包，继续读取剩余部分
		expectedBytes, _ := hex.DecodeString(regPkg)
		if len(expectedBytes) > 1 {
			remainingBytes := make([]byte, len(expectedBytes)-1)
			_, err = io.ReadFull(conn, remainingBytes)
			if err != nil {
				logrus.Warn("读取心跳包剩余数据失败:", err)
				return nil, fmt.Errorf("读取数据失败")
			}

			heartbeatResponse = append(firstByte, remainingBytes...)
			if bytes.Equal(heartbeatResponse, expectedBytes) {
				logrus.Debug("成功读取到心跳包响应")
			}
		}

		// 继续读取 Modbus 响应的第一个字节
		_, err = io.ReadFull(conn, firstByte)
		if err != nil {
			if heartbeatResponse != nil {
				// 如果已经读到心跳包，就返回心跳包
				return heartbeatResponse, nil
			}
			logrus.Warn("读取 Modbus 响应第一个字节失败:", err)
			return nil, fmt.Errorf("读取数据失败")
		}
	}

	// 读取剩余的 MBAP 头部（7字节）
	header := make([]byte, 7)
	_, err = io.ReadFull(conn, header)
	if err != nil {
		if heartbeatResponse != nil {
			// 如果已经读到心跳包，就返回心跳包
			return heartbeatResponse, nil
		}
		logrus.Warn("读取 Modbus TCP 报文头失败:", err)
		return nil, fmt.Errorf("读取报文头失败")
	}

	// 组合完整的头部（8字节）
	fullHeader := append(firstByte, header...)

	// 解析 MBAP 头
	length := binary.BigEndian.Uint16(fullHeader[4:6])
	functionCode := fullHeader[7]

	// 计算需要读取的数据长度
	dataLength := int(length) - 2 // 减去单元ID和功能码长度
	if dataLength < 0 || dataLength > 256 {
		if heartbeatResponse != nil {
			// 如果已经读到心跳包，就返回心跳包
			return heartbeatResponse, nil
		}
		logrus.Warn("无效的数据长度")
		return nil, fmt.Errorf("无效的数据长度")
	}

	// 读取数据部分
	data := make([]byte, dataLength)
	_, err = io.ReadFull(conn, data)
	if err != nil {
		if heartbeatResponse != nil {
			// 如果已经读到心跳包，就返回心跳包
			return heartbeatResponse, nil
		}
		logrus.Warn("读取响应数据失败:", err)
		return nil, fmt.Errorf("读取响应数据失败")
	}

	// 组合完整响应
	modbusResponse := append(fullHeader, data...)
	logrus.Debugf("收到 Modbus 响应: 功能码=0x%02X, 数据长度=%d", functionCode, len(modbusResponse))

	// 如果有心跳包响应，将其与 Modbus 响应组合
	if heartbeatResponse != nil {
		finalResponse := append(heartbeatResponse, modbusResponse...)
		return finalResponse, nil
	}

	return modbusResponse, nil
}
