package services

import (
	"log"
	"net"
	"sync"

	httpclient "github.com/ThingsPanel/modbus-protocol-plugin/http_client"

	globaldata "github.com/ThingsPanel/modbus-protocol-plugin/global_data"
	MQTT "github.com/ThingsPanel/modbus-protocol-plugin/mqtt"
	"github.com/spf13/viper"
)

// 定义全局的conn管道
var connChan = make(chan net.Conn)

func Start() {
	// 启动处理连接的goroutine
	go handleChanConnections()
	// 启动服务
	go startServer()
}

// startServer启动服务
func startServer() {
	serverAddr := viper.GetString("server.address")
	listen, err := net.Listen("tcp", serverAddr)
	if err != nil {
		log.Println("Listen() failed, err: ", err)
		return
	}
	log.Println("modbus服务启动成功：", serverAddr)
	for {
		conn, err := listen.Accept() // 监听客户端的连接请求
		if err != nil {
			log.Println("Accept() failed, err: ", err)
			continue
		}

		// 将接受的conn写入管道
		connChan <- conn
	}
}

// handleConnections处理来自管道的连接
func handleChanConnections() {
	for {
		conn := <-connChan
		go verifyConnection(conn) // 处理每个连接的具体逻辑
	}
}

func CloseConnection(conn net.Conn, token string) {
	err := conn.Close()
	if err != nil {
		log.Println("Close() failed, err: ", err)
	}
	// 删除全局变量
	if _, exists := globaldata.DeviceConnectionMap[token]; !exists {
		return
	} else if conn != *globaldata.DeviceConnectionMap[token] {
		return
	}
	log.Println("删除全局变量完成：", token)
	// 做其他事情，比如发送离线消息
	m := *MQTT.MqttClient
	err = m.SendStatus(token, "0")
	if err != nil {
		log.Println("SendStatus() failed, err: ", err)
	}
	delete(globaldata.GateWayConfigMap, token)
	delete(globaldata.DeviceConnectionMap, token)
	delete(globaldata.GateWayMutexMap, token)
	// 设备离线
	log.Println("设备离线：", token)
}

// 验证连接并继续处理数据
func verifyConnection(conn net.Conn) {
	// 读取客户端发送的数据
	var buf [1024]byte
	n, err := conn.Read(buf[:])
	if err != nil {
		log.Println("Read() failed, err: ", err)
		conn.Close()
		return
	}
	accessToken := string(buf[:n])
	log.Println("收到客户端发来的数据：", accessToken)
	// 首次接收到的是设备token，需要根据token获取设备配置
	// 读取设备配置
	tpGatewayConfig, err := httpclient.GetDeviceConfig(accessToken, "")
	if err != nil {
		// 获取设备配置失败，请检查连接包是否正确
		log.Println("Failed to obtain the device configuration. Please check whether the connection package is correct!", err)
		return
	}
	log.Println("获取设备配置成功：", tpGatewayConfig)
	// 将平台网关的配置存入全局变量
	globaldata.GateWayConfigMap[accessToken] = &tpGatewayConfig.Data
	// 将设备连接存入全局变量
	globaldata.DeviceConnectionMap[accessToken] = &conn
	// 设置锁
	globaldata.GateWayMutexMap[accessToken] = &sync.Mutex{}
	m := *MQTT.MqttClient
	err = m.SendStatus(accessToken, "1")
	if err != nil {
		log.Println("SendStatus() failed, err: ", err)
	}
	// 设备上线
	log.Println("设备上线：", accessToken)
	HandleConn(accessToken) // 处理连接
	// defer conn.Close()
}
