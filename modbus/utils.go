package modbus

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
