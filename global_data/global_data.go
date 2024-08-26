package globaldata

import (
	"encoding/json"
	"strings"
	"sync"

	"github.com/ThingsPanel/tp-protocol-sdk-go/api"
	"github.com/sirupsen/logrus"
)

// 平台网关配置map, key是网关的token，value是网关的配置
// var GateWayConfigMap = make(map[string]*api.DeviceConfigResponseData)
var GateWayConfigMap sync.Map

var SubDeviceConfigMap sync.Map

var SubDeviceIDAndGateWayIDMap sync.Map

// 设备连接map, key是设备的token，value是设备的连接
// var DeviceConnectionMap = make(map[string]*net.Conn)
var DeviceConnectionMap sync.Map

// 设备读写互斥锁
var DeviceRWLock = map[string]*sync.Mutex{}

// modbus错误码映射
var ModbusErrorMap = map[byte]string{
	0x01: "Illegal function(非法功能)",
	0x02: "Illegal data address(非法数据地址)",
	0x03: "Illegal data value(非法数据值)",
	0x04: "Slave device failure(从站设备故障)",
	0x05: "Acknowledge(应答)",
	0x06: "Slave device busy(从站设备忙)",
	0x08: "Memory parity error(存储器奇偶校验错误)",
	0x0A: "Gateway path unavailable(网关路径不可用)",
	0x0B: "Gateway target device failed to respond(网关目标设备未响应)",
}

// modbus错误码方法，返回一个错误说明
func GetModbusErrorDesc(code byte) string {
	if desc, ok := ModbusErrorMap[code]; ok {
		return desc
	}
	return "Unknown error:未知错误"
}

// 通过子设备ID获取网关配置
func GetGateWayConfigByDeviceID(subDeviceID string) (*api.DeviceConfigResponseData, bool) {
	if gateWayID, ok := SubDeviceIDAndGateWayIDMap.Load(subDeviceID); ok {
		if gateWayConfig, ok := GateWayConfigMap.Load(gateWayID); ok {
			return gateWayConfig.(*api.DeviceConfigResponseData), true
		} else {
			logrus.Error("通过网关ID获取网关配置失败")
			return nil, false
		}
	} else {
		logrus.Error("通过子设备ID获取网关ID失败")
		return nil, false
	}
}

// 通过凭证获取regPkg,voucher{"reg_pkg":"` + regPkg + `"}
func GetRegPkgByToken(voucher string) (string, bool) {
	// 去除可能存在的前缀和后缀空白字符
	voucher = strings.TrimSpace(voucher)

	// 检查 voucher 是否为空
	if voucher == "" {
		return "", false
	}

	// 定义一个结构体来解析 JSON
	var data struct {
		RegPkg string `json:"reg_pkg"`
	}

	// 解析 JSON
	err := json.Unmarshal([]byte(voucher), &data)
	if err != nil {
		return "", false
	}

	// 检查 RegPkg 是否为空
	if data.RegPkg == "" {
		return "", false
	}

	return data.RegPkg, true
}
