package services

import (
	"encoding/json"
	"log"
	"net"
	"time"

	globaldata "github.com/ThingsPanel/modbus-protocol-plugin/global_data"
	"github.com/ThingsPanel/modbus-protocol-plugin/modbus"
	MQTT "github.com/ThingsPanel/modbus-protocol-plugin/mqtt"
	tpconfig "github.com/ThingsPanel/modbus-protocol-plugin/tp_config"

	"github.com/ThingsPanel/tp-protocol-sdk-go/api"
)

/*
说明：
硬件设备与平台连接后，一般不会变动配置
所以，不对子设备配置修改做局部更新
而是重新加载网关配置
*/

// HandleConn 处理单个连接
func HandleConn(token string) {

	// 获取网关配置
	m, _ := globaldata.GateWayConfigMap.Load(token)
	gatewayConfig := m.(*api.DeviceConfigResponseData)
	// 遍历网关的子设备
	for _, tpSubDeviceY := range gatewayConfig.SubDevices {
		tpSubDevice:=tpSubDeviceY
		// 将tp子设备的表单配置转SubDeviceFormConfig
		subDeviceFormConfig, err := tpconfig.NewSubDeviceFormConfig(tpSubDevice.Config)
		if err != nil {
			log.Println(err.Error())
			continue
		}
		// 遍历子设备的表单配置
		for _, commandRaw := range subDeviceFormConfig.CommandRawList {
			// 判断网关是ModbusRTU网关还是ModbusTCP网关
			var endianess modbus.EndianessType
			if gatewayConfig.ProtocolType == "MODBUS_RTU" {
				if commandRaw.Endianess == "BIG" {
					endianess = modbus.BigEndian
				} else if commandRaw.Endianess == "LITTLE" {
					endianess = modbus.LittleEndian
				} else {
					// 默认大端
					endianess = modbus.BigEndian
				}
				// 创建RTUCommand
				RTUCommand := modbus.NewRTUCommand(subDeviceFormConfig.SlaveID, commandRaw.FunctionCode, commandRaw.StartingAddress, commandRaw.Quantity, endianess)
				go handleRTUCommand(&RTUCommand, commandRaw, token, &tpSubDevice)

			} else if gatewayConfig.ProtocolType == "MODBUS_TCP" {
				if commandRaw.Endianess == "BIG" {
					endianess = modbus.BigEndian
				} else if commandRaw.Endianess == "LITTLE" {
					endianess = modbus.LittleEndian
				} else {
					// 默认大端
					endianess = modbus.BigEndian
				}
				// 创建TCPCommand
				TCPCommand := modbus.NewTCPCommand(subDeviceFormConfig.SlaveID, commandRaw.FunctionCode, commandRaw.StartingAddress, commandRaw.Quantity, endianess)
				go handleTCPCommand(&TCPCommand, commandRaw, token, &tpSubDevice)
			}
		}
	}
}

func handleRTUCommand(RTUCommand *modbus.RTUCommand, commandRaw *tpconfig.CommandRaw, token string, tpSubDevice *api.SubDevice) {
	data, err := RTUCommand.Serialize()
	if err != nil {
		log.Println(err.Error())
		return
	}

	m, exists := globaldata.DeviceConnectionMap.Load(token)
	if !exists {
		log.Println("No connection found for token:", token)
		return
	}
	gatewayConn := m.(*net.Conn)
	conn := *gatewayConn
	defer CloseConnection(conn, token)

	buf := make([]byte, 1024)

	for {
		if isClose, err := sendRTUDataAndProcessResponse(conn, data, buf, RTUCommand, commandRaw, token, tpSubDevice); err != nil {
			log.Println("Error processing data:", err.Error())
			if isClose {
				return
			}
		}
		// 间隔时间不能小于1秒
		if commandRaw.Interval < 1 {
			commandRaw.Interval = 1
		}
		time.Sleep(time.Duration(commandRaw.Interval) * time.Second)
	}
}
func sendDataAndReadResponse(conn net.Conn, data, buf []byte, token string) (int, error) {
	log.Println("AccessToken:", token, "请求：", data)

	// 设置写超时时间
	err := conn.SetWriteDeadline(time.Now().Add(15 * time.Second))
	if err != nil {
		log.Println("SetWriteDeadline() failed, err: ", err)
		return 0, err
	}

	_, err = conn.Write(data)
	if err != nil {
		return 0, err
	}

	// 设置读取超时时间
	err = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	if err != nil {
		log.Println("SetReadDeadline() failed, err: ", err)
		return 0, err
	}

	n, err := conn.Read(buf)
	if err != nil {
		return n, err
	}
	return n, nil
}

func sendRTUDataAndProcessResponse(conn net.Conn, data, buf []byte, RTUCommand *modbus.RTUCommand, commandRaw *tpconfig.CommandRaw, token string, tpSubDevice *api.SubDevice) (bool, error) {
	n, err := sendDataAndReadResponse(conn, data, buf, token)
	if err != nil {
		return true, err
	}

	log.Println("AccessToken:", token, "返回：", buf[:n])
	respData, err := RTUCommand.ParseAndValidateResponse(buf[:n])
	if err != nil {
		return false, err
	}

	dataMap, err := commandRaw.Serialize(respData)
	if err != nil {
		return false, err
	}

	payloadMap := map[string]interface{}{
		"token":  token,
		"values": map[string]interface{}{tpSubDevice.SubDeviceAddr: dataMap},
	}
	var values []byte
	// 将payloadMap.values 转为json字符串
	values, err = json.Marshal(payloadMap["values"])
	if err != nil {
		return false, err
	}
	payloadMap["values"] = values
	payload, err := json.Marshal(payloadMap)
	if err != nil {
		return false, err
	}

	return false, MQTT.Publish(string(payload))
}

// handleTCPCommand 处理TCPCommand
func handleTCPCommand(TCPCommand *modbus.TCPCommand, commandRaw *tpconfig.CommandRaw, token string, tpSubDevice *api.SubDevice) {
	data, err := TCPCommand.Serialize()
	if err != nil {
		log.Println("Error serializing TCPCommand:", err)
		return
	}

	m, exists := globaldata.DeviceConnectionMap.Load(token)
	if !exists {
		log.Println("No connection found for token:", token)
		return
	}
	gatewayConn := m.(*net.Conn)
	conn := *gatewayConn
	defer CloseConnection(conn, token)

	buf := make([]byte, 1024)

	for {
		if isClose, err := sendTCPDataAndProcessResponse(conn, data, buf, TCPCommand, commandRaw, token, tpSubDevice); err != nil {
			log.Println("Error processing data:", err.Error())
			if isClose {
				return
			}
		}
		time.Sleep(time.Duration(commandRaw.Interval) * time.Second)
	}
}

func sendTCPDataAndProcessResponse(conn net.Conn, data, buf []byte, TCPCommand *modbus.TCPCommand, commandRaw *tpconfig.CommandRaw, token string, tpSubDevice *api.SubDevice) (bool, error) {
	n, err := sendDataAndReadResponse(conn, data, buf, token)
	if err != nil {
		return true, err
	}
	respData, err := TCPCommand.ParseTCPResponse(buf[:n])
	if err != nil {
		return false, err
	}

	dataMap, err := commandRaw.Serialize(respData)
	if err != nil {
		return false, err
	}

	payloadMap := map[string]interface{}{
		"token":  token,
		"values": map[string]interface{}{tpSubDevice.SubDeviceAddr: dataMap},
	}
	var values []byte
	// 将payloadMap.values 转为json字符串
	values, err = json.Marshal(payloadMap["values"])
	if err != nil {
		return false, err
	}
	payloadMap["values"] = values
	payload, err := json.Marshal(payloadMap)
	if err != nil {
		return false, err
	}

	return false, MQTT.Publish(string(payload))
}
