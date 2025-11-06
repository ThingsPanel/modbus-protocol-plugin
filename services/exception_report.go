package services

import (
	"encoding/hex"

	"github.com/sirupsen/logrus"

	"github.com/ThingsPanel/tp-protocol-sdk-go/api"
)

// ReportException 上报所有类型的异常到平台
func ReportException(err error, tpSubDevice *api.SubDevice, rawRequest []byte, rawResponse []byte) {
	if err == nil {
		return
	}

	// 检查是否是ModbusError
	modbusErr, ok := err.(*ModbusError)
	if !ok {
		// 如果不是ModbusError，创建一个Unknown类型的错误
		modbusErr = NewModbusError(ErrorTypeUnknown, 0, err.Error(), err)
	}

	// 构造异常数据
	exceptionData := map[string]interface{}{
		"error_type":    getErrorTypeName(modbusErr.Type),
		"error_message": modbusErr.Message,
	}

	// 所有错误类型都添加原始请求和响应（如果有）
	if len(rawRequest) > 0 {
		exceptionData["raw_request"] = hex.EncodeToString(rawRequest)
	}
	if len(rawResponse) > 0 {
		exceptionData["raw_response"] = hex.EncodeToString(rawResponse)
	}

	// 构造上报数据
	dataMap := map[string]interface{}{
		"modbus_exception": exceptionData,
	}

	// 上报异常信息
	if err := processResponseData(dataMap, tpSubDevice); err != nil {
		logrus.Errorf("Failed to report Modbus exception: %v", err)
	} else {
		logrus.Infof("Reported Modbus exception: type=%s, message=%s", getErrorTypeName(modbusErr.Type), modbusErr.Message)
	}
}

// getErrorTypeName 获取错误类型名称
func getErrorTypeName(errType ErrorType) string {
	switch errType {
	case ErrorTypeConnection:
		return "connection_error"
	case ErrorTypeTimeout:
		return "timeout_error"
	case ErrorTypeBusiness:
		return "business_error"
	case ErrorTypeConfigError:
		return "config_error"
	case ErrorTypeUnknown:
		return "unknown_error"
	default:
		return "unknown_error"
	}
}
