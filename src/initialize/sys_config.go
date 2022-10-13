package initialize

import (
	"log"

	"github.com/spf13/viper"
)

func InitConfigByViper() {
	viper.SetConfigType("yaml")
	viper.SetConfigFile("./config.yaml")
	err := viper.ReadInConfig()
	if err != nil {
		log.Println(err.Error())
	}
}
