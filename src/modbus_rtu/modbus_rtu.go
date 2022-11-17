package modbus_rtu

import (
	"log"
	"time"
	server_map "tp-modbus/map"
	"tp-modbus/src/mqtt"

	"github.com/tbrandon/mbserver"
)

// 启动一个RTU指令携程
func InitRTUGo(gatewayId string, deviceId string) {
	gc := server_map.GatewayChannelMap[gatewayId]
	dc := make(chan int, 1)
	server_map.DeviceChannelSync.Lock()
	server_map.DeviceChannelMap[deviceId] = dc
	server_map.DeviceChannelSync.Unlock()
	var i uint16 = 0
	for {
		if len(gc) > 0 { // 如果通道关闭则跳出携程
			log.Println("网关通道收到信号，设备携程关闭，（设备id:", deviceId, ")")
			break
		}
		if len(dc) > 0 {
			close(dc)
			delete(server_map.DeviceChannelMap, deviceId)
			log.Println("设备通道收到信号，设备携程关闭，（设备id:", deviceId, ")")
			break
		}
		if _, ok := server_map.SubDeviceConfigMap[deviceId]; !ok { //设备被删除
			break
		}
		i++
		var frame mbserver.RTUFrame
		frame.Address = server_map.SubDeviceConfigMap[deviceId].DeviceAddress // 设备地址
		frame.Function = server_map.SubDeviceConfigMap[deviceId].FunctionCode // 功能码
		// 生成指令
		mbserver.SetDataWithRegisterAndNumber(&frame, server_map.SubDeviceConfigMap[deviceId].StartingAddress, server_map.SubDeviceConfigMap[deviceId].AddressNum)
		mqtt.SendMessage(&frame, gatewayId, deviceId, frame.Bytes()) //发送指令给网关设备
		server_map.RTUFrameMap[deviceId] = frame                     //保存子设备指令
		time.Sleep(time.Second * time.Duration(server_map.SubDeviceConfigMap[deviceId].Interval))
	}
}
