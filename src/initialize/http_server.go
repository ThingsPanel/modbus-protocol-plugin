package initialize

import "tp-modbus/src/api"

func InitHttpServer() {
	go api.HttpServer()
}
