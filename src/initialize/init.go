package initialize

import (
	"log"

	"github.com/spf13/viper"
)

func init() {
	InitConfigByViper()
	log.Println("系统配置加载完成...")
	InitHttpServer() //http服务
	log.Println("http服务启动完成...", viper.GetString("http_server.address"))
}
