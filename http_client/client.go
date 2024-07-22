package httpclient

import (
	"fmt"
	"log"
	"time"

	tpprotocolsdkgo "github.com/ThingsPanel/tp-protocol-sdk-go"
	"github.com/ThingsPanel/tp-protocol-sdk-go/api"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var client *tpprotocolsdkgo.Client

func Init() {
	addr := viper.GetString("thingspanel.address")
	logrus.Info("创建http客户端:", addr)
	client = tpprotocolsdkgo.NewClient(addr)
	go ServiceHeartbeat1()
	go ServiceHeartbeat2()
}

func GetDeviceConfig(voucher string, deviceID string) (*api.DeviceConfigResponse, error) {
	deviceConfigReq := api.DeviceConfigRequest{
		Voucher:  voucher,
		DeviceID: deviceID,
	}
	response, err := client.API.GetDeviceConfig(deviceConfigReq)
	if err != nil {
		errMsg := fmt.Sprintf("获取设备配置失败 (请求参数： %+v): %v", deviceConfigReq, err)
		logrus.Info(errMsg)
		return nil, fmt.Errorf(errMsg)
	}
	if response.Code != 200 {
		errMsg := fmt.Sprintf("获取设备配置失败 (请求参数： %+v): %v", deviceConfigReq, response.Message)
		return nil, fmt.Errorf(errMsg)
	}
	return response, nil
}

func ServiceHeartbeat1() {
	for {
		err := reportHeartbeat1()
		if err != nil {
			log.Println(err)
		}
		time.Sleep(50 * time.Second)
	}
}

// 这里需要改为自己的服务
func reportHeartbeat1() error {
	sid := viper.GetString("server.identifier1")
	serviceHeartbeatReq := api.HeartbeatRequest{
		ServiceIdentifier: sid,
	}
	response, err := client.API.Heartbeat(serviceHeartbeatReq)
	if err != nil {
		return fmt.Errorf("服务心跳上报失败 (请求参数：%+v): %v", serviceHeartbeatReq, err)
	}
	if response.Code != 200 {
		return fmt.Errorf("服务心跳上报失败 (请求参数：%+v): %v", serviceHeartbeatReq, response.Message)
	}
	return nil
}
func ServiceHeartbeat2() {
	for {
		err := reportHeartbeat2()
		if err != nil {
			log.Println(err)
		}
		time.Sleep(50 * time.Second)
	}
}

// 这里需要改为自己的服务
func reportHeartbeat2() error {
	sid := viper.GetString("server.identifier2")
	serviceHeartbeatReq := api.HeartbeatRequest{
		ServiceIdentifier: sid,
	}
	response, err := client.API.Heartbeat(serviceHeartbeatReq)
	if err != nil {
		return fmt.Errorf("服务心跳上报失败 (请求参数：%+v): %v", serviceHeartbeatReq, err)
	}
	if response.Code != 200 {
		return fmt.Errorf("服务心跳上报失败 (请求参数：%+v): %v", serviceHeartbeatReq, response.Message)
	}
	return nil
}
