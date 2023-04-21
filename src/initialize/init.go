package initialize

import (
	"log"

	"github.com/spf13/viper"
)

func init() {
	InitHttpServer() //http服务
	log.Println("http服务启动完成...", viper.GetString("http_server.address"))
}
