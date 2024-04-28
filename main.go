package main

import (
	"log"
	"strings"

	//monitor "opcua/examples"
	httpclient "github.com/ThingsPanel/modbus-protocol-plugin/http_client"
	httpserver "github.com/ThingsPanel/modbus-protocol-plugin/http_server"
	mqtt "github.com/ThingsPanel/modbus-protocol-plugin/mqtt"
	deviceconfig "github.com/ThingsPanel/modbus-protocol-plugin/services"
	"github.com/spf13/viper"
)

func main() {
	conf()
	log.Println("Starting the application...")
	LogInIt()
	// 启动mqtt客户端
	mqtt.InitClient()
	// 启动http客户端
	httpclient.Init()
	deviceconfig.Start()
	// 启动http服务
	httpserver.Init()
	// 订阅平台下发的消息
	mqtt.Subscribe()
	select {}
}
func conf() {
	log.Println("加载配置文件...")
	// 设置环境变量前缀
	viper.SetEnvPrefix("MODBUS")
	// 使 Viper 能够读取环境变量
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.SetConfigType("yaml")
	viper.SetConfigFile("./config.yaml")
	err := viper.ReadInConfig()
	if err != nil {
		log.Println(err.Error())
	}
	log.Println("加载配置文件完成...")
}
