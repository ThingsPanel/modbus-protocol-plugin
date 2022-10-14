package tp

import (
	"encoding/json"
	server_map "tp-modbus/map"
	"tp-modbus/src/api"
	"tp-modbus/src/modbus_rtu"
)

// 通过api读取网关设备配置
func GetGatewayConfig(AccessToken string) (server_map.Gateway, error) {

	//subDevice := []Device{device}
	// gateway := server_map.Gateway{
	// 	GatewayId:    "001",
	// 	ProtocolType: "MODBUS_RTU", //MODBUS_RTU MODBUS_TCP
	// 	AccessToken:  "123456",
	// 	SubDevice: []server_map.Device{
	// 		{
	// 			DeviceId:        "001",
	// 			AccessToken:     "654321",
	// 			Interval:        3, //时间间隔
	// 			DeviceAddress:   1, //设备地址
	// 			FunctionCode:    3, //功能码
	// 			StartingAddress: 1, //起始地址
	// 			Key:             "temp",
	// 			AddressNum:      4,
	// 			DataType:        "int64-8",
	// 		},
	// 		{
	// 			DeviceId:        "002",
	// 			AccessToken:     "654322",
	// 			Interval:        3, //时间间隔
	// 			DeviceAddress:   1, //设备地址
	// 			FunctionCode:    3, //功能码
	// 			StartingAddress: 5, //起始地址
	// 			Key:             "temp",
	// 			AddressNum:      4,
	// 			DataType:        "int64-8",
	// 		},
	// 		{
	// 			DeviceId:        "003",
	// 			AccessToken:     "654323",
	// 			Interval:        3, //时间间隔
	// 			DeviceAddress:   1, //设备地址
	// 			FunctionCode:    3, //功能码
	// 			StartingAddress: 9, //起始地址
	// 			Key:             "temp",
	// 			AddressNum:      4,
	// 			DataType:        "int64-8",
	// 		},
	// 		{
	// 			DeviceId:        "004",
	// 			AccessToken:     "654324",
	// 			Interval:        3,  //时间间隔
	// 			DeviceAddress:   1,  //设备地址
	// 			FunctionCode:    3,  //功能码
	// 			StartingAddress: 13, //起始地址
	// 			Key:             "temp",
	// 			AddressNum:      4,
	// 			DataType:        "int64-8",
	// 		},
	// 		{
	// 			DeviceId:        "005",
	// 			AccessToken:     "654325",
	// 			Interval:        3,  //时间间隔
	// 			DeviceAddress:   1,  //设备地址
	// 			FunctionCode:    3,  //功能码
	// 			StartingAddress: 17, //起始地址
	// 			Key:             "temp",
	// 			AddressNum:      4,
	// 			DataType:        "int64-8",
	// 		},
	// 	},
	// }
	var gateway server_map.GatewayData
	var req = make(map[string]interface{})
	req["AccessToken"] = AccessToken
	rsp, err := api.ApiGetGatewayConfig(req)
	if err != nil {
		return gateway.Data, err
	}
	json_error := json.Unmarshal(rsp, &gateway)
	if json_error != nil {
		return gateway.Data, json_error
	}
	server_map.GatewayConfigMap[gateway.Data.GatewayId] = gateway.Data
	for _, subDeviceConfig := range gateway.Data.SubDevice {
		server_map.SubDeviceConfigMap[subDeviceConfig.DeviceId] = subDeviceConfig
	}
	return gateway.Data, nil
}

// 客户端链接成功后启动携程
func ProcessReq(accessToken string) {
	gatewayConfig := server_map.GatewayConfigMap[accessToken]
	if gatewayConfig.ProtocolType == "MODBUS_RTU" {
		for _, deviceConfig := range gatewayConfig.SubDevice {
			go modbus_rtu.InitRTUGo(gatewayConfig.GatewayId, deviceConfig.DeviceId)
		}
	} else if gatewayConfig.ProtocolType == "MODBUS_TCP" {
		for _, deviceConfig := range gatewayConfig.SubDevice {
			go modbus_rtu.InitTCPGo(gatewayConfig.GatewayId, deviceConfig.DeviceId)
		}
	}
}
