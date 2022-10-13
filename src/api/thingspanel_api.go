package api

import "github.com/spf13/viper"

func ApiGetGatewayConfig(req map[string]interface{}) ([]byte, error) {
	response, err := PostJson("http://"+viper.GetString("thingspanel.address")+"/api/gateway/config", req)
	return response, err
}
