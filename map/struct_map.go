package server_map

import "github.com/tbrandon/mbserver"

type Device struct {
	GatewayId       string //网关设备id
	DeviceId        string //子设备id
	AccessToken     string //子设备token
	Interval        int64  //触发时间间隔s
	DeviceAddress   uint8  //设备地址
	FunctionCode    uint8  //功能码
	StartingAddress uint16 //起始地址
	AddressNum      uint16 //地址数量（地址数量返现，根据数据类型后面的数字除以2，比如int32-4的地址数量就是2）
	Key             string //属性名（如：temp,hum等）
	DataType        string //数据类型（3和6功能码的数据类型：int16-2 uint16-2 int32-4 uint32-4 int64-8（一个地址2字节））；uint64在转换中会丢失精度，uint32在转float时候某些值也会丢失精度
	Equation        string
	Precision       string
}

type Gateway struct {
	Id           string   //网关设备id
	ProtocolType string   //modbus协议类型：RTU TCP
	AccessToken  string   //网关设备token
	SubDevice    []Device //子设备
}
type GatewayData struct {
	Code    int     `json:"code"`
	Message string  `json:"message"`
	Data    Gateway `json:"data"`
}

var GatewayConfigMap = make(map[string]Gateway)      // 网关tcp客户端集合
var SubDeviceConfigMap = make(map[string]Device)     // 子设备配置集合
var TCPFrameMap = make(map[string]mbserver.TCPFrame) // 子设备请求指令集合
var RTUFrameMap = make(map[string]mbserver.RTUFrame) // 子设备请求指令集合
