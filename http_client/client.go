package httpclient

import (
	"fmt"
	"log"

	tpprotocolsdkgo "github.com/ThingsPanel/tp-protocol-sdk-go"
	"github.com/ThingsPanel/tp-protocol-sdk-go/api"
	"github.com/spf13/viper"
)

var client *tpprotocolsdkgo.Client

func Init() {
	addr := viper.GetString("thingspanel.address")
	log.Println("创建http客户端:", addr)
	client = tpprotocolsdkgo.NewClient(addr)
}

func GetDeviceConfig(accessToken string, deviceID string) (*api.DeviceConfigResponse, error) {
	deviceConfigReq := api.DeviceConfigRequest{
		AccessToken: accessToken,
		DeviceID:    deviceID,
	}
	response, err := client.API.GetDeviceConfig(deviceConfigReq)
	if err != nil {
		errMsg := fmt.Sprintf("获取设备配置失败 (请求参数： %+v): %v", deviceConfigReq, err)
		log.Println(errMsg)
		return nil, fmt.Errorf(errMsg)
	}
	return response, nil
}
