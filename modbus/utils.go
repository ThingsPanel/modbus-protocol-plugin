package modbus

// ParseModbusExceptionResponse 解析Modbus异常响应
// 返回 (isException, exceptionCode, functionCode)
func ParseModbusExceptionResponse(data []byte, modbusType string) (bool, byte, byte) {
	if len(data) < 3 {
		return false, 0, 0
	}

	var functionCode byte
	var exceptionCode byte

	if modbusType == "TCP" {
		// TCP格式: [事务ID1, 事务ID2, 协议ID1, 协议ID2, 长度1, 长度2, 单元ID, 功能码|0x80, 异常码]
		if len(data) >= 9 {
			functionCode = data[7]
			if functionCode&0x80 != 0 {
				exceptionCode = data[8]
				return true, exceptionCode, functionCode & 0x7F
			}
		}
	} else {
		// RTU格式: [地址, 功能码|0x80, 异常码, CRC1, CRC2]
		if len(data) >= 3 {
			functionCode = data[1]
			if functionCode&0x80 != 0 {
				exceptionCode = data[2]
				return true, exceptionCode, functionCode & 0x7F
			}
		}
	}

	return false, 0, 0
}

func crc16(data []byte) uint16 {
	const polynomial = 0xA001
	var crc = uint16(0xFFFF)

	for _, byteVal := range data {
		crc ^= uint16(byteVal)
		for i := 0; i < 8; i++ {
			if (crc & 0x0001) != 0 {
				crc = (crc >> 1) ^ polynomial
			} else {
				crc >>= 1
			}
		}
	}
	return crc
}
