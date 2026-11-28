package services

import (
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	globaldata "github.com/ThingsPanel/modbus-protocol-plugin/global_data"
	"github.com/ThingsPanel/modbus-protocol-plugin/modbus"
	"github.com/ThingsPanel/modbus-protocol-plugin/mqtt"
	tpconfig "github.com/ThingsPanel/modbus-protocol-plugin/tp_config"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/ThingsPanel/tp-protocol-sdk-go/api"
)

// HandleConn 处理单个连接
func HandleConn(regPkg, deviceID string, active *net.Conn) {
	// 获取网关配置
	m, _ := globaldata.GateWayConfigMap.Load(deviceID)
	gatewayConfig := m.(*api.DeviceConfigResponseData)

	// 遍历网关的子设备
	for _, tpSubDevice := range gatewayConfig.SubDevices {
		// 存储子设备配置
		globaldata.SubDeviceConfigMap.Store(tpSubDevice.DeviceID, &tpSubDevice)
		globaldata.SubDeviceIDAndGateWayIDMap.Store(tpSubDevice.DeviceID, deviceID)

		// 将tp子设备的表单配置转SubDeviceFormConfig
		subDeviceFormConfig, err := tpconfig.NewSubDeviceFormConfig(tpSubDevice.ProtocolConfigTemplate, tpSubDevice.SubDeviceAddr)
		if err != nil {
			logrus.Error(err.Error())
			continue
		}

		// 遍历子设备的表单配置
		for _, commandRaw := range subDeviceFormConfig.CommandRawList {
			var endianess modbus.EndianessType
			if gatewayConfig.ProtocolType == "MODBUS_RTU" {
				switch commandRaw.Endianess {
				case "BIG":
					endianess = modbus.BigEndian
				case "LITTLE":
					endianess = modbus.LittleEndian
				case "BADC":
					endianess = modbus.ByteSwap
				case "CDAB":
					endianess = modbus.WordByteSwap
				default:
					endianess = modbus.BigEndian
				}
				cmd := modbus.NewRTUCommand(subDeviceFormConfig.SlaveID, commandRaw.FunctionCode, commandRaw.StartingAddress, commandRaw.Quantity, endianess)
				go handleRTUCommandLoop(&cmd, commandRaw, &tpSubDevice, gatewayConfig, active)

			} else if gatewayConfig.ProtocolType == "MODBUS_TCP" {
				switch commandRaw.Endianess {
				case "BIG":
					endianess = modbus.BigEndian
				case "LITTLE":
					endianess = modbus.LittleEndian
				case "BADC":
					endianess = modbus.ByteSwap
				case "CDAB":
					endianess = modbus.WordByteSwap
				default:
					endianess = modbus.BigEndian
				}
				cmd := modbus.NewTCPCommand(subDeviceFormConfig.SlaveID, commandRaw.FunctionCode, commandRaw.StartingAddress, commandRaw.Quantity, endianess)
				go handleTCPCommandLoop(&cmd, commandRaw, regPkg, &tpSubDevice, gatewayConfig, active)
			}
		}
	}
}

// handleRTUCommandLoop RTU命令循环处理
func handleRTUCommandLoop(cmd *modbus.RTUCommand, commandRaw *tpconfig.CommandRaw, subDevice *api.SubDevice, gateway *api.DeviceConfigResponseData, active *net.Conn) {
	deviceID := gateway.ID
	data, err := cmd.Serialize()
	if err != nil {
		logrus.Error(err.Error())
		return
	}

	if commandRaw.Interval < 1 {
		commandRaw.Interval = 1
	}
	interval := time.Duration(commandRaw.Interval) * time.Second

	logrus.Infof("RTU命令循环启动: deviceID=%s, 功能码=0x%02X", deviceID, cmd.FunctionCode)
	lock := globaldata.DeviceLock(deviceID)

	for {
		lock.Lock()
		connVal, exists := globaldata.DeviceConnectionMap.Load(deviceID)
		if !exists || connVal != active {
			lock.Unlock()
			return
		}
		conn := *active

		// 发送并处理响应
		err := sendRTUDataAndProcessResponse(conn, data, cmd, commandRaw, subDevice, gateway)

		lock.Unlock()

		if err != nil {
			// 任何错误都断开连接，让设备重连
			logrus.Warnf("RTU错误，断开连接: deviceID=%s, error=%s", deviceID, err.Error())
			CloseConnection(conn, deviceID)
			return
		}

		// 等待间隔时间
		time.Sleep(interval)
	}
}

// handleTCPCommandLoop TCP命令循环处理
func handleTCPCommandLoop(cmd *modbus.TCPCommand, commandRaw *tpconfig.CommandRaw, regPkg string, subDevice *api.SubDevice, gateway *api.DeviceConfigResponseData, active *net.Conn) {
	deviceID := gateway.ID
	data, err := cmd.Serialize()
	if err != nil {
		logrus.Error(err.Error())
		return
	}

	if commandRaw.Interval < 1 {
		commandRaw.Interval = 1
	}
	interval := time.Duration(commandRaw.Interval) * time.Second

	logrus.Infof("TCP命令循环启动: deviceID=%s, 功能码=0x%02X", deviceID, cmd.FunctionCode)
	lock := globaldata.DeviceLock(deviceID)

	for {
		lock.Lock()
		connVal, exists := globaldata.DeviceConnectionMap.Load(deviceID)
		if !exists || connVal != active {
			lock.Unlock()
			return
		}
		conn := *active

		// 发送并处理响应
		err := sendTCPDataAndProcessResponse(conn, data, cmd, commandRaw, regPkg, subDevice, gateway)

		lock.Unlock()

		if err != nil {
			// 任何错误都断开连接，让设备重连
			logrus.Warnf("TCP错误，断开连接: deviceID=%s, error=%s", deviceID, err.Error())
			CloseConnection(conn, deviceID)
			return
		}

		// 等待间隔时间
		time.Sleep(interval)
	}
}

// sendRTUDataAndProcessResponse 发送RTU数据并处理响应
func sendRTUDataAndProcessResponse(conn net.Conn, data []byte, cmd *modbus.RTUCommand, commandRaw *tpconfig.CommandRaw, subDevice *api.SubDevice, gateway *api.DeviceConfigResponseData) error {
	// 清空缓冲区
	clearBuffer(conn)

	// 写入数据
	err := conn.SetWriteDeadline(time.Now().Add(3 * time.Second))
	if err != nil {
		return err
	}
	_, err = conn.Write(data)
	if err != nil {
		return err
	}

	// 读取响应
	err = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	if err != nil {
		return err
	}

	buf, err := ReadModbusRTUResponse(conn, cmd.FunctionCode)
	if err != nil {
		return err
	}

	if len(buf) == 0 {
		return NewModbusError(ErrorTypeTimeout, 0, "Read timeout", nil)
	}

	// 检查Modbus异常响应
	isException, exceptionCode, functionCode := modbus.ParseModbusExceptionResponse(buf, "RTU")
	if isException {
		desc := globaldata.GetModbusErrorDesc(exceptionCode)
		errMsg := fmt.Sprintf("Modbus exception: func=0x%02X, code=0x%02X, %s", functionCode, exceptionCode, desc)
		err := NewModbusError(ErrorTypeBusiness, exceptionCode, errMsg, nil)
		ReportException(err, subDevice, data, buf)
		return err
	}

	// 解析响应
	respData, err := cmd.ParseAndValidateResponse(buf)
	if err != nil {
		ReportException(err, subDevice, data, buf)
		return err
	}

	// 序列化数据
	dataMap, err := commandRaw.Serialize(respData)
	if err != nil {
		ReportException(err, subDevice, data, buf)
		return err
	}

	return mqtt.PublishCollectedTelemetry(gateway, subDevice, dataMap)
}

// sendTCPDataAndProcessResponse 发送TCP数据并处理响应
func sendTCPDataAndProcessResponse(conn net.Conn, data []byte, cmd *modbus.TCPCommand, commandRaw *tpconfig.CommandRaw, regPkg string, subDevice *api.SubDevice, gateway *api.DeviceConfigResponseData) error {
	// 清空缓冲区
	clearBuffer(conn)

	// 写入数据
	err := conn.SetWriteDeadline(time.Now().Add(3 * time.Second))
	if err != nil {
		return err
	}
	_, err = conn.Write(data)
	if err != nil {
		return err
	}

	// 读取响应
	err = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	if err != nil {
		return err
	}

	buf, err := ReadModbusTCPResponse(conn, regPkg)
	if err != nil {
		return err
	}

	// 检查Modbus异常响应
	isException, exceptionCode, functionCode := modbus.ParseModbusExceptionResponse(buf, "TCP")
	if isException {
		desc := globaldata.GetModbusErrorDesc(exceptionCode)
		errMsg := fmt.Sprintf("Modbus exception: func=0x%02X, code=0x%02X, %s", functionCode, exceptionCode, desc)
		err := NewModbusError(ErrorTypeBusiness, exceptionCode, errMsg, nil)
		ReportException(err, subDevice, data, buf)
		return err
	}

	// 解析响应
	respData, err := cmd.ParseTCPResponse(buf)
	if err != nil {
		ReportException(err, subDevice, data, buf)
		return err
	}

	// 序列化数据
	dataMap, err := commandRaw.Serialize(respData)
	if err != nil {
		ReportException(err, subDevice, data, buf)
		return err
	}

	return mqtt.PublishCollectedTelemetry(gateway, subDevice, dataMap)
}

// clearBuffer 清空连接缓冲区
func clearBuffer(conn net.Conn) error {
	buf := make([]byte, 1024)
	for {
		err := conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
		if err != nil {
			if errors.Is(err, net.ErrClosed) || strings.Contains(err.Error(), "use of closed network connection") {
				return nil
			}
			return err
		}

		_, err = conn.Read(buf)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				return nil
			}
			if errors.Is(err, net.ErrClosed) || strings.Contains(err.Error(), "use of closed network connection") {
				return nil
			}
			return err
		}
	}
}

// flushTimeoutResponse 排空超时后的迟到响应
func flushTimeoutResponse(conn net.Conn) {
	enabled := viper.GetBool("flush_mechanism.enabled")
	if !enabled {
		return
	}

	silencePeriod := viper.GetDuration("flush_mechanism.silence_period")
	if silencePeriod <= 0 {
		silencePeriod = 100 * time.Millisecond
	}

	logrus.Debugf("开始排空迟到响应，静默期=%v", silencePeriod)

	buf := make([]byte, 256)
	lastReadTime := time.Now()
	totalDropped := 0

	for {
		err := conn.SetReadDeadline(time.Now().Add(50 * time.Millisecond))
		if err != nil {
			if errors.Is(err, net.ErrClosed) || strings.Contains(err.Error(), "use of closed network connection") {
				return
			}
			return
		}

		n, err := conn.Read(buf)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				silenceDuration := time.Since(lastReadTime)
				if silenceDuration >= silencePeriod {
					logrus.Debugf("排空完成，共丢弃 %d 字节", totalDropped)
					return
				}
				continue
			}
			if errors.Is(err, net.ErrClosed) || strings.Contains(err.Error(), "use of closed network connection") {
				return
			}
			return
		}

		if n > 0 {
			totalDropped += n
			logrus.Debugf("排空: 丢弃 %d 字节", n)
			lastReadTime = time.Now()
		}

		if time.Since(lastReadTime) > 1*time.Second && totalDropped == 0 {
			return
		}
	}
}
