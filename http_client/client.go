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
var managed *managedPlatform

func Init() error {
	addr := viper.GetString("thingspanel.address")
	if viper.GetBool("thingspanel.managed") {
		var err error
		managed, err = newManagedPlatform(addr, viper.GetString("thingspanel.token"))
		if err != nil {
			return err
		}
	} else {
		client = tpprotocolsdkgo.NewClient(addr)
	}
	go ServiceHeartbeat1()
	go ServiceHeartbeat2()
	return nil
}

func GetDeviceConfig(voucher string, deviceID string) (*api.DeviceConfigResponse, error) {
	deviceConfigReq := api.DeviceConfigRequest{
		Voucher:  voucher,
		DeviceID: deviceID,
	}
	if managed != nil {
		var response api.DeviceConfigResponse
		if err := managed.post("/api/v1/plugin/device/config", deviceConfigReq, &response); err != nil {
			return nil, err
		}
		if response.Code != 200 || response.Data.ID == "" ||
			(response.Data.ProtocolType != "MODBUS_TCP" && response.Data.ProtocolType != "MODBUS_RTU") {
			return nil, fmt.Errorf("managed gateway configuration was rejected")
		}
		return &response, nil
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
	return reportHeartbeat(viper.GetString("server.identifier1"))
}

func reportHeartbeat(sid string) error {
	serviceHeartbeatReq := api.HeartbeatRequest{
		ServiceIdentifier: sid,
	}
	if managed != nil {
		var response api.HeartbeatResponseData
		if err := managed.post("/api/v1/plugin/heartbeat", serviceHeartbeatReq, &response); err != nil {
			return err
		}
		if response.Code != 200 {
			return fmt.Errorf("managed heartbeat rejected for %s", sid)
		}
		return nil
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
	return reportHeartbeat(viper.GetString("server.identifier2"))
}
