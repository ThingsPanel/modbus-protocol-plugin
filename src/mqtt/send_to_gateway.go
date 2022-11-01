package mqtt

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"log"
	server_map "tp-modbus/map"
	"tp-modbus/src/util"

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
	if server_map.SubDeviceConfigMap[deviceId].FunctionCode == uint8(3) {
		RspRtuReadHoldingRegisters(frame, buf[:n], deviceId)
	}

}

// 功能码-3，读保持寄存器
func RspRtuReadHoldingRegisters(frame mbserver.Framer, data []byte, deviceId string) {
	if res := bytes.Compare(frame.Bytes()[0:2], data[0:2]); res == 0 { // 正常返回
		b := data[3 : len(data)-2]
		var payloadMap = make(map[string]interface{})
		var valueMap = make(map[string]interface{})
		valueMap[server_map.SubDeviceConfigMap[deviceId].Key] = BytesToInt(b, server_map.SubDeviceConfigMap[deviceId].DataType)
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
		log.Println("网关设备异常返回:", data)
	}
}

//2字节转int64
func BytesToInt(b []byte, dataType string) interface{} {
	bytesBuffer := bytes.NewBuffer(b)
	switch dataType {
	case "int16-2":
		var x int16
		binary.Read(bytesBuffer, binary.BigEndian, &x)
		return int64(x)
	case "int32-4":
		var x int32
		binary.Read(bytesBuffer, binary.BigEndian, &x)
		return int64(x)
	case "int64-8":
		var x int64
		binary.Read(bytesBuffer, binary.BigEndian, &x)
		return int64(x)
	case "uint16-2":
		var x uint16
		binary.Read(bytesBuffer, binary.BigEndian, &x)
		return int64(x)
	case "uint32-4":
		var x uint32
		binary.Read(bytesBuffer, binary.BigEndian, &x)
		return int64(x)
	case "float64-8":
		return util.Float64frombytes(b)
	default:
		return int64(0)
	}
}
