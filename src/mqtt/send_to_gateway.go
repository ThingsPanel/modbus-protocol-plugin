package mqtt

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"log"
	"strings"
	server_map "tp-modbus/map"
	"tp-modbus/src/util"

	"github.com/gogf/gf/encoding/gbinary"
	"github.com/tbrandon/mbserver"
)

// 发送指令给设备网关
func SendMessage(frame mbserver.Framer, gatewayId string, deviceId string, message []byte) {
	// 发送指令时候抢占锁，等待接受完后解锁
	log.Println("发送指令给网关设备（id:", gatewayId, "）：", message)
	server_map.TcpClientSyncMap[gatewayId].Lock()
	server_map.TcpClientMap[gatewayId].Write(message)
	reader := bufio.NewReader(server_map.TcpClientMap[gatewayId])
	var buf [128]byte
	n, err := reader.Read(buf[:]) // 读取数据
	server_map.TcpClientSyncMap[gatewayId].Unlock()
	if err != nil {
		log.Println("网关设备(id:" + gatewayId + ")已断开连接!")
		server_map.CloseGatewayGoroutine(gatewayId) // 关闭网关携程
	}
	//recvStr := string(buf[:n])
	log.Println("收到网关设备（id:", gatewayId, "）发来的数据：", buf[:n])
	// 判断功能码
	if server_map.SubDeviceConfigMap[deviceId].FunctionCode == uint8(3) {
		if server_map.GatewayConfigMap[gatewayId].ProtocolType == "MODBUS_RTU" {
			RspRtuReadHoldingRegisters(frame, buf[:n], deviceId)
		} else if server_map.GatewayConfigMap[gatewayId].ProtocolType == "MODBUS_TCP" {
			RspTcpReadHoldingRegisters(frame, buf[:n], deviceId)
		}
	} else if server_map.SubDeviceConfigMap[deviceId].FunctionCode == uint8(1) {
		if server_map.GatewayConfigMap[gatewayId].ProtocolType == "MODBUS_RTU" {
			RspReadCoils(frame, buf[:n], deviceId)
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

// RTU功能码-1，读保持寄存器
// frame为发送指令的结构体
// 解析出数据数据
func RspReadCoils(frame mbserver.Framer, data []byte, deviceId string) {
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
	// 别名数组的数量和值的数量必须相等且不等于0
	if len(keyList) == len(valueList) && (len(keyList) != 0 && len(valueList) != 0) {
		for index, key := range keyList {
			valueMap[key] = valueList[index]
		}
		payloadMap["token"] = server_map.SubDeviceConfigMap[deviceId].AccessToken
		payloadMap["values"] = valueMap
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
		payloadMap["values"] = valueMap
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
	case "float64-8":
		for i := 0; i < len(b)/8; i++ {
			v_list = append(v_list, util.Float64frombytes(b[i*8:i*8+8]))
		}
		return v_list
	default:
		return v_list
	}
}
