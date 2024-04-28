package httpclient

import (
	"fmt"

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
