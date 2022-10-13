package server_map

import "sync"

var GatewayChannelMap = make(map[string]chan int) // 网关通道集合
var DeviceChannelMap = make(map[string]chan int)  // 设备通道集合
var DeviceChannelSync sync.Mutex                  // 设备通道互斥锁（保证网关下设备在创建通道的时候不会出现并发异常）

// 关闭网关通道
func CloseGatewayGoroutine(gatewayId string) {
	GatewayChannelMap[gatewayId] <- 1
}

// 关闭设备通道
func CloseDeviceGoroutine(deviceId string) {
	DeviceChannelMap[deviceId] <- 1
}
