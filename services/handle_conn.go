package services

import (
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"time"

	globaldata "github.com/ThingsPanel/modbus-protocol-plugin/global_data"
	"github.com/ThingsPanel/modbus-protocol-plugin/modbus"
	MQTT "github.com/ThingsPanel/modbus-protocol-plugin/mqtt"
	tpconfig "github.com/ThingsPanel/modbus-protocol-plugin/tp_config"
	"github.com/sirupsen/logrus"

	"github.com/ThingsPanel/tp-protocol-sdk-go/api"
)

/*
说明：
硬件设备与平台连接后，一般不会变动配置
所以，不对子设备配置修改做局部更新
而是重新加载网关配置
*/

// HandleConn 处理单个连接
func HandleConn(regPkg, deviceID string) {
	// 获取网关配置
	m, _ := globaldata.GateWayConfigMap.Load(deviceID)
	gatewayConfig := m.(*api.DeviceConfigResponseData)
	// 遍历网关的子设备
	for _, tpSubDevice := range gatewayConfig.SubDevices {
		// 存储子设备配置
		globaldata.SubDeviceConfigMap.Store(tpSubDevice.DeviceID, &tpSubDevice)
		// 存储子设备id和网关id的映射关系
		globaldata.SubDeviceIDAndGateWayIDMap.Store(tpSubDevice.DeviceID, deviceID)
		// 将tp子设备的表单配置转SubDeviceFormConfig
		subDeviceFormConfig, err := tpconfig.NewSubDeviceFormConfig(tpSubDevice.ProtocolConfigTemplate, tpSubDevice.SubDeviceAddr)
		if err != nil {
			logrus.Info(err.Error())
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
				go handleRTUCommand(&RTUCommand, commandRaw, regPkg, &tpSubDevice, deviceID)

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
				go handleTCPCommand(&TCPCommand, commandRaw, regPkg, &tpSubDevice, deviceID)
			}
		}
	}
}

// 开启线程处理RTUCommand
func handleRTUCommand(RTUCommand *modbus.RTUCommand, commandRaw *tpconfig.CommandRaw, regPkg string, tpSubDevice *api.SubDevice, deviceID string) {
	data, err := RTUCommand.Serialize()
	if err != nil {
		logrus.Info(err.Error())
		return
	}

	m, exists := globaldata.DeviceConnectionMap.Load(deviceID)
	if !exists {
		logrus.Info("No connection found for regPkg:", regPkg, " deviceID:", deviceID)
		return
	}
	gatewayConn := m.(*net.Conn)
	conn := *gatewayConn
	defer CloseConnection(conn, deviceID)

	for {
		isClose, err := sendRTUDataAndProcessResponse(conn, data, RTUCommand, commandRaw, regPkg, tpSubDevice)
		if err != nil {
			logrus.Info("Error processing data:", err.Error())
			if isClose {
				conn.Close()
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

// clearBuffer 清空连接缓冲区
func clearBuffer(conn net.Conn) error {
	buf := make([]byte, 1024)
	for {
		// 设置读取超时时间为100ms
		err := conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
		if err != nil {
			return err
		}

		_, err = conn.Read(buf)
		if err != nil {
			// 如果是超时错误，说明缓冲区已清空
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				return nil
			}
			return err
		}
	}
}

func sendDataAndReadResponse(conn net.Conn, data []byte, regPkg string, modbusType string) (int, []byte, error) {
	// 获取锁
	if _, exists := globaldata.DeviceRWLock[regPkg]; !exists {
		globaldata.DeviceRWLock[regPkg] = &sync.Mutex{}
	}
	globaldata.DeviceRWLock[regPkg].Lock()
	logrus.Info("获取到锁：", regPkg)
	defer globaldata.DeviceRWLock[regPkg].Unlock()
	logrus.Info("regPkg:", regPkg, " 请求：", data)

	// 清空缓冲区
	if err := clearBuffer(conn); err != nil {
		logrus.Info("清空缓冲区失败:", err)
	}

	// 设置写超时时间
	err := conn.SetWriteDeadline(time.Now().Add(1 * time.Second))
	if err != nil {
		logrus.Info("SetWriteDeadline() failed, err: ", err)
	}

	// 写入数据
	_, err = conn.Write(data)
	if err != nil {
		return 0, nil, err
	}

	// 设置读取超时时间
	err = conn.SetReadDeadline(time.Now().Add(1 * time.Second))
	if err != nil {
		logrus.Info("SetReadDeadline() failed, err: ", err)
	}

	var buf []byte
	if modbusType == "RTU" {
		buf, err = ReadModbusRTUResponse(conn, regPkg)
		if err != nil {
			return 0, nil, err
		}
	} else if modbusType == "TCP" {
		buf, err = ReadModbusTCPResponse(conn, regPkg)
		if err != nil {
			return 0, nil, err
		}
	} else {
		return 0, nil, fmt.Errorf("unsupported modbus type")
	}
	n := len(buf)
	logrus.Info("regPkg:", regPkg, " 返回：", buf[:n])
	return n, buf, nil
}

func sendRTUDataAndProcessResponse(conn net.Conn, data []byte, RTUCommand *modbus.RTUCommand, commandRaw *tpconfig.CommandRaw, regPkg string, tpSubDevice *api.SubDevice) (bool, error) {
	n, buf, err := sendDataAndReadResponse(conn, data, regPkg, "RTU")
	if err != nil {
		if err.Error() == "not supported function code" || err.Error() == "read failed" {
			return false, err
		}
		return true, err
	}

	respData, err := RTUCommand.ParseAndValidateResponse(buf[:n])
	if err != nil {
		return false, err
	}

	dataMap, err := commandRaw.Serialize(respData)
	if err != nil {
		return false, err
	}

	payloadMap := map[string]interface{}{
		"device_id": tpSubDevice.DeviceID,
		"values":    dataMap,
	}
	var values []byte
	// 将payloadMap.values 转为json字符串
	values, err = json.Marshal(payloadMap["values"])
	if err != nil {
		return false, err
	}
	logrus.Info("values:", string(values))
	payloadMap["values"] = values
	payload, err := json.Marshal(payloadMap)
	if err != nil {
		return false, err
	}

	return false, MQTT.Publish(string(payload))
}

// handleTCPCommand 处理TCPCommand
func handleTCPCommand(TCPCommand *modbus.TCPCommand, commandRaw *tpconfig.CommandRaw, regPkg string, tpSubDevice *api.SubDevice, deviceID string) {
	data, err := TCPCommand.Serialize()
	if err != nil {
		logrus.Info("Error serializing TCPCommand:", err)
		return
	}

	m, exists := globaldata.DeviceConnectionMap.Load(deviceID)
	if !exists {
		logrus.Info("No connection found for regPkg:", regPkg, " deviceID:", deviceID)
		return
	}
	gatewayConn := m.(*net.Conn)
	conn := *gatewayConn
	defer CloseConnection(conn, deviceID)

	for {
		if isClose, err := sendTCPDataAndProcessResponse(conn, data, TCPCommand, commandRaw, regPkg, tpSubDevice); err != nil {
			if isClose {
				conn.Close()
				return
			}
		}
		time.Sleep(time.Duration(commandRaw.Interval) * time.Second)
	}
}

func sendTCPDataAndProcessResponse(conn net.Conn, data []byte, TCPCommand *modbus.TCPCommand, commandRaw *tpconfig.CommandRaw, regPkg string, tpSubDevice *api.SubDevice) (bool, error) {
	n, buf, err := sendDataAndReadResponse(conn, data, regPkg, "TCP")
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
		"device_id": tpSubDevice.DeviceID,
		//"values": map[string]interface{}{tpSubDevice.SubDeviceAddr: dataMap},
		"values": dataMap,
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

// 开启线程处理RTUCommand
func OneHandleRTUCommand(RTUCommand *modbus.RTUCommand, commandRaw *tpconfig.CommandRaw, regPkg string, tpSubDevice *api.SubDevice, deviceID string) {
	data, err := RTUCommand.Serialize()
	if err != nil {
		logrus.Info(err.Error())
		return
	}

	m, exists := globaldata.DeviceConnectionMap.Load(deviceID)
	if !exists {
		logrus.Info("No connection found for regPkg:", regPkg, " deviceID:", deviceID)
		return
	}
	gatewayConn := m.(*net.Conn)
	conn := *gatewayConn
	defer CloseConnection(conn, deviceID)

	for {
		isClose, err := sendRTUDataAndProcessResponse(conn, data, RTUCommand, commandRaw, regPkg, tpSubDevice)
		if err != nil {
			logrus.Info("Error processing data:", err.Error())
			if isClose {
				conn.Close()
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
