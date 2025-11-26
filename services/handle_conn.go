package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	globaldata "github.com/ThingsPanel/modbus-protocol-plugin/global_data"
	"github.com/ThingsPanel/modbus-protocol-plugin/modbus"
	MQTT "github.com/ThingsPanel/modbus-protocol-plugin/mqtt"
	tpconfig "github.com/ThingsPanel/modbus-protocol-plugin/tp_config"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"

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
				} else if commandRaw.Endianess == "BADC" {
					endianess = modbus.ByteSwap
				} else if commandRaw.Endianess == "CDAB" {
					endianess = modbus.WordByteSwap
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
				} else if commandRaw.Endianess == "BADC" {
					endianess = modbus.ByteSwap
				} else if commandRaw.Endianess == "CDAB" {
					endianess = modbus.WordByteSwap
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
	// 注意：不要在这里使用 defer CloseConnection，因为多个goroutine共享同一个连接
	// 单个goroutine退出不应该关闭共享的连接

	// 确保间隔时间不小于1秒
	if commandRaw.Interval < 1 {
		commandRaw.Interval = 1
	}
	interval := time.Duration(commandRaw.Interval) * time.Second

	for {
		shouldClose, err := sendRTUDataAndProcessResponse(conn, data, RTUCommand, commandRaw, regPkg, tpSubDevice)

		if err != nil {
			modbusErr := ClassifyError(err)
			// 业务错误（Modbus异常响应）继续运行，不关闭连接
			if modbusErr != nil && modbusErr.IsBusinessError() {
				logrus.Debugf("业务错误，继续运行: %s", err.Error())
			} else {
				logrus.Infof("处理数据时出错: %s", err.Error())
			}

			// 如果需要关闭连接，则关闭连接并退出循环
			if shouldClose {
				logrus.Warnf("连接需要关闭，关闭连接并退出处理循环: regPkg=%s, deviceID=%s, error=%s", regPkg, deviceID, err.Error())
				// 这里使用 deviceID 作为全局映射的 key（与 verifyConnection 中保存的一致）
				CloseConnection(conn, deviceID)
				return
			}

			// 检查连接是否仍然有效（在 sleep 之前检查，避免无效连接继续运行）
			if modbusErr != nil && modbusErr.Type == ErrorTypeConnection {
				// 连接错误，即使 shouldClose 为 false，也应该关闭连接并退出循环
				logrus.Warnf("检测到连接错误，关闭连接并退出处理循环: regPkg=%s, deviceID=%s, error=%s", regPkg, deviceID, err.Error())
				CloseConnection(conn, deviceID)
				return
			}
		}

		time.Sleep(interval)
	}
}

// clearBuffer 清空连接缓冲区
func clearBuffer(conn net.Conn) error {
	buf := make([]byte, 1024)
	for {
		// 设置读取超时时间为100ms
		err := conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
		if err != nil {
			// 如果连接已关闭，静默处理（连接已关闭时清空缓冲区没有意义）
			// 检查是否是连接关闭错误（兼容多种错误格式）
			if errors.Is(err, net.ErrClosed) || strings.Contains(err.Error(), "use of closed network connection") {
				return nil
			}
			return err
		}

		_, err = conn.Read(buf)
		if err != nil {
			// 如果是超时错误，说明缓冲区已清空
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				return nil
			}
			// 如果连接已关闭，静默处理（连接已关闭时清空缓冲区没有意义）
			// 检查是否是连接关闭错误（兼容多种错误格式）
			if errors.Is(err, net.ErrClosed) || strings.Contains(err.Error(), "use of closed network connection") {
				return nil
			}
			return err
		}
	}
}

// flushTimeoutResponse 排空超时后的迟到响应
// 持续读取并丢弃数据，直到静默期超过配置的时间
func flushTimeoutResponse(conn net.Conn) {
	// 检查是否启用排空机制
	enabled := viper.GetBool("flush_mechanism.enabled")
	if !enabled {
		logrus.Debug("排空机制未启用，跳过排空")
		return
	}

	// 获取静默期时间
	silencePeriod := viper.GetDuration("flush_mechanism.silence_period")
	if silencePeriod <= 0 {
		silencePeriod = 100 * time.Millisecond // 默认值
	}

	logrus.Debugf("开始排空超时后的迟到响应，静默期=%v", silencePeriod)

	buf := make([]byte, 256)
	lastReadTime := time.Now()
	totalDropped := 0

	for {
		// 设置较短的读取超时，用于检测是否有数据
		err := conn.SetReadDeadline(time.Now().Add(50 * time.Millisecond))
		if err != nil {
			// 连接已关闭，停止排空
			if errors.Is(err, net.ErrClosed) || strings.Contains(err.Error(), "use of closed network connection") {
				logrus.Debug("排空过程中连接已关闭，停止排空")
				return
			}
			logrus.Warn("设置读取超时失败:", err)
			return
		}

		n, err := conn.Read(buf)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				// 超时，检查静默期
				silenceDuration := time.Since(lastReadTime)
				if silenceDuration >= silencePeriod {
					// 静默期已超过配置时间，排空完成
					logrus.Debugf("排空完成，静默期=%v >= 配置时间=%v，共丢弃 %d 字节", silenceDuration, silencePeriod, totalDropped)
					return
				}
				// 继续等待
				continue
			}
			// 连接关闭错误，停止排空
			if errors.Is(err, net.ErrClosed) || strings.Contains(err.Error(), "use of closed network connection") {
				logrus.Debug("排空过程中连接已关闭，停止排空")
				return
			}
			// 其他错误，停止排空
			logrus.Warn("排空过程中发生错误:", err)
			return
		}

		// 读取到数据，丢弃并重置静默期计时
		if n > 0 {
			totalDropped += n
			logrus.Debugf("排空: 丢弃 %d 字节数据（累计: %d 字节）", n, totalDropped)
			lastReadTime = time.Now()
		}

		// 检查总排空时间，避免无限等待
		if time.Since(lastReadTime) > 1*time.Second && totalDropped == 0 {
			logrus.Debug("排空超时且无数据，结束")
			return
		}
	}
}

// sendDataAndReadResponseOnce 执行一次数据发送和读取（不包含重试）
func sendDataAndReadResponseOnce(conn net.Conn, data []byte, regPkg string, modbusType string) (int, []byte, *ModbusError) {
	// 清空缓冲区
	if err := clearBuffer(conn); err != nil {
		logrus.Info("清空缓冲区失败:", err)
	}

	// 设置写超时时间
	err := conn.SetWriteDeadline(time.Now().Add(1 * time.Second))
	if err != nil {
		// 如果连接已关闭，立即返回错误
		return 0, nil, ClassifyError(err)
	}

	// 写入数据
	_, err = conn.Write(data)
	if err != nil {
		return 0, nil, ClassifyError(err)
	}

	// 设置读取超时时间
	err = conn.SetReadDeadline(time.Now().Add(1 * time.Second))
	if err != nil {
		// 如果连接已关闭，立即返回错误
		return 0, nil, ClassifyError(err)
	}

	var buf []byte
	var readTimeout bool
	if modbusType == "RTU" {
		buf, err = ReadModbusRTUResponse(conn, regPkg)
		// RTU在超时时返回nil, nil，需要检查buf是否为nil
		if err != nil {
			return 0, nil, ClassifyError(err)
		}
		if len(buf) == 0 {
			readTimeout = true
		}
	} else if modbusType == "TCP" {
		buf, err = ReadModbusTCPResponse(conn, regPkg)
		// TCP在超时时返回错误，检查是否为超时错误
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				readTimeout = true
			} else {
				return 0, nil, ClassifyError(err)
			}
		}
	} else {
		return 0, nil, NewModbusError(ErrorTypeConfigError, 0, "unsupported modbus type", nil)
	}

	// 如果读取超时，调用排空机制清理迟到响应
	if readTimeout {
		logrus.Info("读取超时，开始排空迟到响应")
		flushTimeoutResponse(conn)
		return 0, nil, NewModbusError(ErrorTypeTimeout, 0, "Read response timeout", nil)
	}

	// 检查是否是Modbus异常响应
	isException, exceptionCode, functionCode := modbus.ParseModbusExceptionResponse(buf, modbusType)
	if isException {
		desc := globaldata.GetModbusErrorDesc(exceptionCode)
		errMsg := fmt.Sprintf("Modbus exception response: function_code=0x%02X, exception_code=0x%02X, %s", functionCode, exceptionCode, desc)
		err := NewModbusError(ErrorTypeBusiness, exceptionCode, errMsg, nil)
		err.FunctionCode = functionCode
		// 返回实际的响应数据，以便上报时包含原始响应
		n := len(buf)
		return n, buf, err
	}

	n := len(buf)
	logrus.Info("regPkg:", regPkg, " 返回：", buf[:n])
	return n, buf, nil
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

	// 检查是否启用重试机制
	retryEnabled := viper.GetBool("retry_mechanism.enabled")
	maxRetries := viper.GetInt("retry_mechanism.max_retries")
	retryInterval := viper.GetDuration("retry_mechanism.retry_interval")
	backoffMultiplier := viper.GetFloat64("retry_mechanism.backoff_multiplier")

	if !retryEnabled || maxRetries <= 0 {
		// 不启用重试，直接执行一次
		n, buf, err := sendDataAndReadResponseOnce(conn, data, regPkg, modbusType)
		if err != nil {
			// 如果是业务错误（Modbus异常响应），返回响应数据以便上报
			if err.IsBusinessError() {
				return n, buf, err
			}
			return 0, nil, err
		}
		return n, buf, nil
	}

	// 启用重试机制
	var lastErr *ModbusError
	var lastBuf []byte
	var lastN int
	currentInterval := retryInterval

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			logrus.Infof("重试第 %d 次，等待 %v 后重试", attempt, currentInterval)
			time.Sleep(currentInterval)
			// 指数退避
			currentInterval = time.Duration(float64(currentInterval) * backoffMultiplier)
		}

		n, buf, err := sendDataAndReadResponseOnce(conn, data, regPkg, modbusType)
		if err == nil {
			// 成功，返回结果
			if attempt > 0 {
				logrus.Infof("重试成功，在第 %d 次重试后成功", attempt)
			}
			return n, buf, nil
		}

		lastErr = err
		// 如果是业务错误（Modbus异常响应），保存响应数据以便上报
		if err.IsBusinessError() {
			lastBuf = buf
			lastN = n
			logrus.Infof("业务错误（Modbus异常响应）: %s，停止重试并返回响应数据", err.Error())
			return lastN, lastBuf, err
		}

		// 检查错误是否可重试
		if !err.IsRetryable() {
			logrus.Infof("错误不可重试: %s，停止重试", err.Error())
			return 0, nil, err
		}

		// 如果是最后一次尝试，不再重试
		if attempt >= maxRetries {
			logrus.Warnf("达到最大重试次数 %d，放弃重试", maxRetries)
			break
		}

		logrus.Warnf("第 %d 次尝试失败: %s，将进行重试", attempt+1, err.Error())
	}

	// 所有重试都失败
	return 0, nil, lastErr
}

// handleModbusError 统一处理Modbus错误
// 返回 (shouldCloseConnection, error)
func handleModbusError(err error) (bool, error) {
	if err == nil {
		return false, nil
	}
	modbusErr := ClassifyError(err)
	shouldClose, isBusinessError := modbusErr.ShouldCloseConnection()

	if isBusinessError {
		// 业务错误（Modbus异常响应）记录日志但继续运行，不关闭连接
		logrus.Warnf("Modbus异常响应（业务错误）: %s，继续运行", err.Error())
		return false, err
	}

	return shouldClose, err
}

// processResponseData 处理响应数据并发布到MQTT
func processResponseData(dataMap map[string]interface{}, tpSubDevice *api.SubDevice) error {
	payloadMap := map[string]interface{}{
		"device_id": tpSubDevice.DeviceID,
		"values":    dataMap,
	}

	// 将payloadMap.values 转为json字符串
	values, err := json.Marshal(payloadMap["values"])
	if err != nil {
		return err
	}
	logrus.Info("values:", string(values))

	payloadMap["values"] = values
	payload, err := json.Marshal(payloadMap)
	if err != nil {
		return err
	}

	return MQTT.Publish(string(payload))
}

func sendRTUDataAndProcessResponse(conn net.Conn, data []byte, RTUCommand *modbus.RTUCommand, commandRaw *tpconfig.CommandRaw, regPkg string, tpSubDevice *api.SubDevice) (bool, error) {
	// 1. 发送数据并读取响应
	n, buf, err := sendDataAndReadResponse(conn, data, regPkg, "RTU")
	if err != nil {
		// 上报所有类型的异常
		var rawResponse []byte
		if buf != nil && n > 0 {
			rawResponse = buf[:n]
		}
		ReportException(err, tpSubDevice, data, rawResponse)
		return handleModbusError(err)
	}

	// 2. 解析响应
	respData, err := RTUCommand.ParseAndValidateResponse(buf[:n])
	if err != nil {
		// 上报解析响应时的异常，包含原始响应
		ReportException(err, tpSubDevice, data, buf[:n])
		return false, err
	}

	// 3. 序列化数据
	dataMap, err := commandRaw.Serialize(respData)
	if err != nil {
		// 检查是否是业务错误（Modbus异常响应），如果是则上报异常，包含原始响应
		modbusErr := ClassifyError(err)
		if modbusErr.Type == ErrorTypeBusiness {
			ReportException(err, tpSubDevice, data, buf[:n])
		}
		return handleModbusError(err)
	}

	// 4. 发布到MQTT
	return false, processResponseData(dataMap, tpSubDevice)
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
	// 注意：不要在这里使用 defer CloseConnection，因为多个goroutine共享同一个连接
	// 单个goroutine退出不应该关闭共享的连接

	// 确保间隔时间不小于1秒
	if commandRaw.Interval < 1 {
		commandRaw.Interval = 1
	}
	interval := time.Duration(commandRaw.Interval) * time.Second

	for {
		shouldClose, err := sendTCPDataAndProcessResponse(conn, data, TCPCommand, commandRaw, regPkg, tpSubDevice)

		if err != nil {
			modbusErr := ClassifyError(err)
			// 业务错误（Modbus异常响应）继续运行，不关闭连接
			if modbusErr != nil && modbusErr.IsBusinessError() {
				logrus.Debugf("业务错误，继续运行: %s", err.Error())
			} else {
				logrus.Infof("处理数据时出错: %s", err.Error())
			}

			// 如果需要关闭连接，则关闭连接并退出循环
			if shouldClose {
				logrus.Warnf("连接需要关闭，关闭连接并退出处理循环: regPkg=%s, deviceID=%s, error=%s", regPkg, deviceID, err.Error())
				// 这里使用 deviceID 作为全局映射的 key（与 verifyConnection 中保存的一致）
				CloseConnection(conn, deviceID)
				return
			}
		}

		// 检查连接是否仍然有效（在 sleep 之前检查，避免无效连接继续运行）
		if err != nil {
			// 如果发生错误且不是业务错误，检查连接状态
			modbusErr := ClassifyError(err)
			if modbusErr != nil && modbusErr.Type == ErrorTypeConnection {
				// 连接错误，即使 shouldClose 为 false，也应该关闭连接并退出循环
				logrus.Warnf("检测到连接错误，关闭连接并退出处理循环: regPkg=%s, deviceID=%s, error=%s", regPkg, deviceID, err.Error())
				CloseConnection(conn, deviceID)
				return
			}
		}

		time.Sleep(interval)
	}
}

func sendTCPDataAndProcessResponse(conn net.Conn, data []byte, TCPCommand *modbus.TCPCommand, commandRaw *tpconfig.CommandRaw, regPkg string, tpSubDevice *api.SubDevice) (bool, error) {
	// 1. 发送数据并读取响应
	n, buf, err := sendDataAndReadResponse(conn, data, regPkg, "TCP")
	if err != nil {
		// 上报所有类型的异常
		var rawResponse []byte
		if buf != nil && n > 0 {
			rawResponse = buf[:n]
		}
		ReportException(err, tpSubDevice, data, rawResponse)
		return handleModbusError(err)
	}

	// 2. 解析响应
	respData, err := TCPCommand.ParseTCPResponse(buf[:n])
	if err != nil {
		// 上报解析响应时的异常，包含原始响应
		ReportException(err, tpSubDevice, data, buf[:n])
		return false, err
	}

	// 3. 序列化数据
	dataMap, err := commandRaw.Serialize(respData)
	if err != nil {
		// 检查是否是业务错误（Modbus异常响应），如果是则上报异常，包含原始响应
		modbusErr := ClassifyError(err)
		if modbusErr.Type == ErrorTypeBusiness {
			ReportException(err, tpSubDevice, data, buf[:n])
		}
		return handleModbusError(err)
	}

	// 4. 发布到MQTT
	return false, processResponseData(dataMap, tpSubDevice)
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
	// 注意：不要在这里使用 defer CloseConnection，因为多个goroutine共享同一个连接
	// 单个goroutine退出不应该关闭共享的连接

	// 确保间隔时间不小于1秒
	if commandRaw.Interval < 1 {
		commandRaw.Interval = 1
	}
	interval := time.Duration(commandRaw.Interval) * time.Second

	for {
		shouldClose, err := sendRTUDataAndProcessResponse(conn, data, RTUCommand, commandRaw, regPkg, tpSubDevice)

		if err != nil {
			modbusErr := ClassifyError(err)
			// 业务错误（Modbus异常响应）继续运行，不关闭连接
			if modbusErr != nil && modbusErr.IsBusinessError() {
				logrus.Debugf("业务错误，继续运行: %s", err.Error())
			} else {
				logrus.Infof("处理数据时出错: %s", err.Error())
			}

			// 如果需要关闭连接，则退出循环
			// 注意：不在这里关闭连接，因为多个goroutine共享同一个连接
			// 连接应该由连接管理逻辑统一管理（在services.go的verifyConnection中）
			if shouldClose {
				logrus.Warnf("连接需要关闭，退出处理循环: regPkg=%s, error=%s", regPkg, err.Error())
				return
			}

			// 检查连接是否仍然有效（在 sleep 之前检查，避免无效连接继续运行）
			if modbusErr != nil && modbusErr.Type == ErrorTypeConnection {
				// 连接错误，即使 shouldClose 为 false，也应该退出循环
				logrus.Warnf("检测到连接错误，退出处理循环: regPkg=%s, error=%s", regPkg, err.Error())
				return
			}
		}

		time.Sleep(interval)
	}
}
