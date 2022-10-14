package api

import (
	"os"

	"github.com/spf13/viper"
)

func ApiGetGatewayConfig(req map[string]interface{}) ([]byte, error) {
	TpHost := os.Getenv("TP_HOST")
	if TpHost == "" {
		TpHost = viper.GetString("thingspanel.address")
	}
	response, err := PostJson("http://"+TpHost+"/api/gateway/config", req)
	return response, err
}
