package main

import (
	"log"
	"os"
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
	if err := httpclient.Init(); err != nil {
		log.Fatal(err)
	}
	mqtt.InitClient()
	if err := deviceconfig.Start(); err != nil {
		log.Fatal(err)
	}
	// 订阅平台下发的消息
	mqtt.Subscribe()
	log.Fatal(httpserver.Serve())
}
func conf() {
	log.Println("加载配置文件...")
	// 设置环境变量前缀
	viper.SetEnvPrefix("MODBUS")
	// 使 Viper 能够读取环境变量
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.SetConfigType("yaml")
	configFile := os.Getenv("MODBUS_CONFIG_FILE")
	if configFile == "" {
		configFile = "./config.yaml"
	}
	viper.SetConfigFile(configFile)
	err := viper.ReadInConfig()
	if err != nil {
		if viper.GetBool("thingspanel.managed") {
			log.Fatal(err)
		}
		log.Println(err.Error())
	}
	log.Println("加载配置文件完成...")
}
