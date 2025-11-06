package services

import (
	"errors"
	"fmt"
	"net"
	"strings"

	globaldata "github.com/ThingsPanel/modbus-protocol-plugin/global_data"
)

// ErrorType 错误类型
type ErrorType int

const (
	ErrorTypeConnection  ErrorType = iota // 连接错误（需要关闭连接）
	ErrorTypeTimeout                      // 超时错误（可重试）
	ErrorTypeBusiness                     // 业务错误（Modbus异常响应，不关闭连接）
	ErrorTypeConfigError                  // 配置错误
	ErrorTypeUnknown                      // 未知错误
)

// ModbusError Modbus错误封装
type ModbusError struct {
	Type         ErrorType
	Code         byte   // Modbus异常码（如果是业务错误）
	FunctionCode byte   // Modbus功能码（如果是业务错误）
	Message      string // 错误消息
	OriginalErr  error  // 原始错误
}

// Error 实现error接口
func (e *ModbusError) Error() string {
	if e.OriginalErr != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.OriginalErr)
	}
	return e.Message
}

// IsBusinessError 判断是否为业务错误（Modbus异常响应）
func (e *ModbusError) IsBusinessError() bool {
	return e.Type == ErrorTypeBusiness
}

// IsRetryable 判断是否可重试
func (e *ModbusError) IsRetryable() bool {
	// 超时错误可重试，业务错误不可重试，连接错误不可重试
	return e.Type == ErrorTypeTimeout
}

// ShouldCloseConnection 判断是否需要关闭连接
// 返回 (shouldClose, isBusinessError)
func (e *ModbusError) ShouldCloseConnection() (bool, bool) {
	if e.Type == ErrorTypeBusiness {
		// 业务错误不关闭连接
		return false, true
	}
	if e.Type == ErrorTypeConnection {
		// 连接错误需要关闭连接
		return true, false
	}
	// 其他错误不关闭连接（超时、配置错误等）
	return false, false
}

// NewModbusError 创建新的Modbus错误
func NewModbusError(errType ErrorType, code byte, message string, originalErr error) *ModbusError {
	return &ModbusError{
		Type:        errType,
		Code:        code,
		Message:     message,
		OriginalErr: originalErr,
	}
}

// ClassifyError 分类错误
func ClassifyError(err error) *ModbusError {
	if err == nil {
		return nil
	}

	// 如果已经是 ModbusError，直接返回（避免重复分类导致类型丢失）
	if modbusErr, ok := err.(*ModbusError); ok {
		return modbusErr
	}

	errStr := err.Error()

	// 优先检查是否是网络错误（使用errors.As解包错误链，处理被fmt.Errorf包装的错误）
	var netErr net.Error
	if errors.As(err, &netErr) {
		if netErr.Timeout() {
			return NewModbusError(ErrorTypeTimeout, 0, "Read response timeout", err)
		}
		// 其他网络错误（包括连接断开、连接重置等）都视为连接错误
		// TCP连接断开时，conn.Read()/conn.Write()会返回网络错误，Go会自动处理跨平台差异
		return NewModbusError(ErrorTypeConnection, 0, "Network connection error", err)
	}

	// 检查是否是连接关闭错误
	if errors.Is(err, net.ErrClosed) {
		return NewModbusError(ErrorTypeConnection, 0, "Connection closed", err)
	}

	// 检查是否是Modbus异常响应（业务错误）
	// 格式: "function Code(0xXX) exception Code(0xYY):描述"
	if strings.Contains(errStr, "function Code") && strings.Contains(errStr, "exception Code") {
		// 提取异常码
		var code byte
		var functionCode byte
		fmt.Sscanf(errStr, "function Code(0x%02x) exception Code(0x%02x)", &functionCode, &code)
		desc := globaldata.GetModbusErrorDesc(code)
		modbusErr := NewModbusError(ErrorTypeBusiness, code, fmt.Sprintf("Modbus exception response: function_code=0x%02X, exception_code=0x%02X, %s", functionCode, code, desc), err)
		modbusErr.FunctionCode = functionCode
		return modbusErr
	}

	// 检查是否是已知的业务错误
	if errStr == "not supported function code" || errStr == "read failed" {
		return NewModbusError(ErrorTypeBusiness, 0, errStr, err)
	}

	// 未知错误
	return NewModbusError(ErrorTypeUnknown, 0, errStr, err)
}
