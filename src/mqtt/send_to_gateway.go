package mqtt

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"log"
	"strings"
	"time"
	server_map "tp-modbus/map"
	"tp-modbus/src/util"

	"github.com/spf13/cast"

	"math/rand"

	"github.com/gogf/gf/encoding/gbinary"
	"github.com/shopspring/decimal"
	"github.com/tbrandon/mbserver"
)

// 发送指令给设备网关
func SendMessage(frame mbserver.Framer, gatewayId string, deviceId string, message []byte) {
	rand.Seed(time.Now().UnixNano())
	number := rand.Intn(10000)
	// 发送指令时候抢占锁，等待接受完后解锁
	server_map.TcpClientSyncMap[gatewayId].Lock()
	log.Println(number, "加锁，发送====》设备(id:", gatewayId, "):", message)
	server_map.TcpClientMap[gatewayId].Write(message)
	reader := bufio.NewReader(server_map.TcpClientMap[gatewayId])
	var buf [1024]byte
	n, err := reader.Read(buf[:]) // 读取数据

	if err != nil {
		SendStatus(server_map.GatewayConfigMap[gatewayId].AccessToken, "0")
		log.Println("网关设备(id:" + gatewayId + ")已断开连接!")
		server_map.CloseGatewayGoroutine(gatewayId) // 关闭网关携程
	}
	//recvStr := string(buf[:n])
	log.Println(number, "解锁，接收《====设备(id:", gatewayId, "):", buf[:n])
	
	server_map.TcpClientSyncMap[gatewayId].Unlock()

	// 判断功能码
	if frame.GetFunction() == uint8(3) {
		if server_map.GatewayConfigMap[gatewayId].ProtocolType == "MODBUS_RTU" {
			RspRtuReadHoldingRegisters(frame, buf[:n], deviceId)
		} else if server_map.GatewayConfigMap[gatewayId].ProtocolType == "MODBUS_TCP" {
			RspTcpReadHoldingRegisters(frame, buf[:n], deviceId)
		}
	} else if frame.GetFunction() == uint8(1) {
		if server_map.GatewayConfigMap[gatewayId].ProtocolType == "MODBUS_RTU" {
			RspRTUReadCoils(frame, buf[:n], deviceId)
		} else if server_map.GatewayConfigMap[gatewayId].ProtocolType == "MODBUS_TCP" {
			RspTCPReadCoils(frame, buf[:n], deviceId)
		}
	} else if frame.GetFunction() == uint8(2) {
		if server_map.GatewayConfigMap[gatewayId].ProtocolType == "MODBUS_RTU" {
			RspRTUReadDiscreteInputs(frame, buf[:n], deviceId)
		} else if server_map.GatewayConfigMap[gatewayId].ProtocolType == "MODBUS_TCP" {
			RspTCPReadDiscreteInputs(frame, buf[:n], deviceId)
		}
	} else if frame.GetFunction() == uint8(4) {
		if server_map.GatewayConfigMap[gatewayId].ProtocolType == "MODBUS_RTU" {
			RspRtuReadInputRegisters(frame, buf[:n], deviceId)
		} else if server_map.GatewayConfigMap[gatewayId].ProtocolType == "MODBUS_TCP" {
			RspTcpReadInputRegisters(frame, buf[:n], deviceId)
		}
	}
}

// RTU功能码-3，读保持寄存器
// frame为发送指令的结构体
// 解析出数据数据
func RspRtuReadHoldingRegisters(frame mbserver.Framer, data []byte, deviceId string) {
	// 返回功能码是3为正常，81为异常
	if res := bytes.Compare(frame.Bytes()[0:2], data[0:2]); res == 0 { // 正常返回
		b := data[3 : len(data)-2] // 数值
		BytesAnalysisAndSend(b, deviceId)
	} else {
		log.Println("网关设备异常返回:", data)
	}
}

// RTU功能码-4，读输入寄存器
// frame为发送指令的结构体
// 解析出数据数据
func RspRtuReadInputRegisters(frame mbserver.Framer, data []byte, deviceId string) {
	// 返回功能码是4为正常，81为异常
	if res := bytes.Compare(frame.Bytes()[0:2], data[0:2]); res == 0 { // 正常返回
		b := data[3 : len(data)-2] // 数值
		BytesAnalysisAndSend(b, deviceId)
	} else {
		log.Println("网关设备异常返回:", data)
	}
}

// TCP功能码-4，读输入寄存器
// frame为发送指令的结构体
// 解析出数据数据
func RspTcpReadInputRegisters(frame mbserver.Framer, data []byte, deviceId string) {
	// 判断返回的地址码和功能码是否一致
	if res := bytes.Compare(frame.Bytes()[6:8], data[6:8]); res == 0 {
		var b_len = uint8(data[8])
		b := data[9 : 9+b_len]
		BytesAnalysisAndSend(b, deviceId)
	} else {
		log.Println("网关设备异常返回:", data)
	}
}

// TCP功能码-3，读保持寄存器
// frame为发送指令的结构体
// 解析出数据数据
func RspTcpReadHoldingRegisters(frame mbserver.Framer, data []byte, deviceId string) {
	// 判断返回的地址码和功能码是否一致
	if res := bytes.Compare(frame.Bytes()[6:8], data[6:8]); res == 0 {
		var b_len = uint8(data[8])
		b := data[9 : 9+b_len]
		BytesAnalysisAndSend(b, deviceId)
	} else {
		log.Println("网关设备异常返回:", data)
	}
}

// RTU功能码-1，读线圈状态
// frame为发送指令的结构体
// 解析出数据数据
func RspRTUReadCoils(frame mbserver.Framer, data []byte, deviceId string) {
	// 返回功能码是1为正常，81为异常
	if res := bytes.Compare(frame.Bytes()[0:2], data[0:2]); res == 0 { // 正常返回
		b := data[3 : len(data)-2] // 数值
		BytesAnalysisAndSend1(b, deviceId)
	} else {
		log.Println("网关设备异常返回:", data)
	}
}

// TCP功能码-1，读线圈状态
// frame为发送指令的结构体
// 解析出数据数据
func RspTCPReadCoils(frame mbserver.Framer, data []byte, deviceId string) {
	// 返回功能码是1为正常，81为异常
	if res := bytes.Compare(frame.Bytes()[6:8], data[6:8]); res == 0 {
		var b_len = uint8(data[8])
		b := data[9 : 9+b_len]
		BytesAnalysisAndSend1(b, deviceId)
	} else {
		log.Println("网关设备异常返回:", data)
	}
}

// TCP功能码-2，读输入位状态
// frame为发送指令的结构体
// 解析出数据数据
func RspTCPReadDiscreteInputs(frame mbserver.Framer, data []byte, deviceId string) {
	// 返回功能码是1为正常，81为异常
	if res := bytes.Compare(frame.Bytes()[6:8], data[6:8]); res == 0 {
		var b_len = uint8(data[8])
		b := data[9 : 9+b_len]
		BytesAnalysisAndSend1(b, deviceId)
	} else {
		log.Println("网关设备异常返回:", data)
	}
}

// RTU功能码-2，读输入位状态
// frame为发送指令的结构体
// 解析出数据数据
func RspRTUReadDiscreteInputs(frame mbserver.Framer, data []byte, deviceId string) {
	// 返回功能码是1为正常，81为异常
	if res := bytes.Compare(frame.Bytes()[0:2], data[0:2]); res == 0 { // 正常返回
		b := data[3 : len(data)-2] // 数值
		BytesAnalysisAndSend1(b, deviceId)
	} else {
		log.Println("网关设备异常返回:", data)
	}
}

//功能码3字节解析和发送
func BytesAnalysisAndSend(b []byte, deviceId string) {
	var payloadMap = make(map[string]interface{})
	var valueMap = make(map[string]interface{})
	keyList := strings.Split(server_map.SubDeviceConfigMap[deviceId].Key, ",")
	valueList := BytesToInt(b, server_map.SubDeviceConfigMap[deviceId].DataType)
	EquationList := strings.Split(server_map.SubDeviceConfigMap[deviceId].Equation, ",")
	PrecisionList := strings.Split(server_map.SubDeviceConfigMap[deviceId].Precision, ",")
	// 别名数组的数量和值的数量必须相等且不等于0
	if len(keyList) == len(valueList) && (len(keyList) != 0 && len(valueList) != 0) {
		for index, key := range keyList {
			//判断公式
			if len(EquationList) == 1 {
				if EquationList[0] != "" {
					value, err := util.Equation(EquationList[0], valueList[index])
					if err != nil {
						log.Println("公式执行错误：", err.Error())
					}
					valueMap[key] = value
				} else {
					valueMap[key] = valueList[index]
				}
			} else {
				if EquationList[index] != "" {
					value, err := util.Equation(EquationList[index], valueList[index])
					if err != nil {
						log.Println("公式执行错误：", err.Error())
					}
					valueMap[key] = value
				} else {
					valueMap[key] = valueList[index]
				}

			}
			//判断精度
			if len(PrecisionList) == 1 {
				if PrecisionList[0] != "" {
					if value, ok := valueMap[key].(float64); ok {
						valueMap[key], _ = decimal.NewFromFloat(value).Round(cast.ToInt32(PrecisionList[0])).Float64()
					}
				}
			} else {
				if EquationList[index] != "" {
					if value, ok := valueMap[key].(float64); ok {
						valueMap[key], _ = decimal.NewFromFloat(value).Round(cast.ToInt32(PrecisionList[index])).Float64()
					}
				}
			}

		}
		payloadMap["token"] = server_map.SubDeviceConfigMap[deviceId].AccessToken
		log.Println("发送的values:", valueMap)
		valueByte, err := json.Marshal(valueMap)
		if err != nil {
			log.Println("map转json格式错误...", err.Error(), valueMap)
		}
		payloadMap["values"] = valueByte
		log.Println(payloadMap)
		payload, err := json.Marshal(payloadMap)
		if err != nil {
			log.Println("map转json格式错误...", err.Error(), payloadMap)
		} else {
			Send(payload)
		}
	} else {
		log.Println("别名数组的数量和值的数量不一致！")
	}
}

//功能码1字节解析和发送
// 87654321 高位在前，低位在后
func BytesAnalysisAndSend1(b []byte, deviceId string) {
	var payloadMap = make(map[string]interface{})
	var valueMap = make(map[string]interface{})
	keyList := strings.Split(server_map.SubDeviceConfigMap[deviceId].Key, ",")
	//valueList := BytesToInt(b, server_map.SubDeviceConfigMap[deviceId].DataType)
	// 返回的线圈字节数应该满足配置的key列表长度
	bit := gbinary.DecodeBytesToBits(b) //转bit数组
	log.Println(bit)
	if len(keyList) <= len(bit) {
		for index, key := range keyList {
			valueMap[key] = bit[len(bit)-1-index]
		}
		payloadMap["token"] = server_map.SubDeviceConfigMap[deviceId].AccessToken
		log.Println("发送的values:", valueMap)
		valueByte, err := json.Marshal(valueMap)
		if err != nil {
			log.Println("map转json格式错误...", err.Error(), valueMap)
		}
		payloadMap["values"] = valueByte
		log.Println(payloadMap)
		payload, err := json.Marshal(payloadMap)
		if err != nil {
			log.Println("map转json格式错误...", err.Error(), payloadMap)
		} else {
			Send(payload)
		}
	} else {
		log.Println("别名数组的数量和值的数量不一致！")
	}
}

//字节转数值
func BytesToInt(b []byte, dataType string) []interface{} {
	var v_list []interface{}
	switch dataType {
	case "int16-2":
		for i := 0; i < len(b)/2; i++ {
			var v int16
			bytesBuffer := bytes.NewBuffer(b[i*2 : i*2+2])
			binary.Read(bytesBuffer, binary.BigEndian, &v)
			v_list = append(v_list, v)
		}
		return v_list
	case "int32-4":
		for i := 0; i < len(b)/4; i++ {
			var v int32
			bytesBuffer := bytes.NewBuffer(b[i*4 : i*4+4])
			binary.Read(bytesBuffer, binary.BigEndian, &v)
			v_list = append(v_list, v)
		}
		return v_list
	case "int64-8":
		for i := 0; i < len(b)/8; i++ {
			var v int64
			bytesBuffer := bytes.NewBuffer(b[i*8 : i*8+8])
			binary.Read(bytesBuffer, binary.BigEndian, &v)
			v_list = append(v_list, v)
		}
		return v_list
	case "uint16-2":
		for i := 0; i < len(b)/2; i++ {
			var v uint16
			bytesBuffer := bytes.NewBuffer(b[i*2 : i*2+2])
			binary.Read(bytesBuffer, binary.BigEndian, &v)
			v_list = append(v_list, v)
		}
		return v_list
	case "uint32-4":
		for i := 0; i < len(b)/4; i++ {
			var v uint32
			bytesBuffer := bytes.NewBuffer(b[i*4 : i*4+4])
			binary.Read(bytesBuffer, binary.BigEndian, &v)
			v_list = append(v_list, v)
		}
		return v_list
	case "float32-4":
		for i := 0; i < len(b)/4; i++ {
			v_list = append(v_list, util.Float32frombytes(b[i*4:i*4+4]))
		}
		return v_list
	case "float64-8":
		for i := 0; i < len(b)/8; i++ {
			v_list = append(v_list, util.Float64frombytes(b[i*8:i*8+8]))
		}
		return v_list
	default:
		return v_list
	}
}
