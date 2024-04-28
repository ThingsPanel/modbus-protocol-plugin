package services

import (
	"net"

	httpclient "github.com/ThingsPanel/modbus-protocol-plugin/http_client"
	"github.com/sirupsen/logrus"

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
		logrus.Info("Listen() failed, err: ", err)
		return
	}
	logrus.Info("modbus服务启动成功：", serverAddr)
	for {
		conn, err := listen.Accept() // 监听客户端的连接请求
		if err != nil {
			logrus.Info("Accept() failed, err: ", err)
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
		logrus.Info("Close() failed, err: ", err)
	}
	// 删除全局变量
	if m, exists := globaldata.DeviceConnectionMap.Load(token); !exists {
		return
	} else if conn != *m.(*net.Conn) {
		return
	}
	logrus.Info("删除全局变量完成：", token)
	// 做其他事情，比如发送离线消息
	m := *MQTT.MqttClient
	err = m.SendStatus(token, "0")
	if err != nil {
		logrus.Info("SendStatus() failed, err: ", err)
	}
	globaldata.GateWayConfigMap.Delete(token)
	globaldata.DeviceConnectionMap.Delete(token)
	// 设备离线
	logrus.Info("设备离线：", token)
}

// 验证连接并继续处理数据
func verifyConnection(conn net.Conn) {
	// 读取客户端发送的数据
	var buf [1024]byte
	n, err := conn.Read(buf[:])
	if err != nil {
		logrus.Info("Read() failed, err: ", err)
		conn.Close()
		return
	}
	regPkg := string(buf[:n])
	logrus.Info("收到客户端发来的注册包：", regPkg)
	// 首次接收到的是设备regPkg，需要根据regPkg获取设备配置
	// 凭借voucher
	voucher := `{"reg_pkg":"` + regPkg + `"}`
	// 读取设备配置
	tpGatewayConfig, err := httpclient.GetDeviceConfig(voucher, "")
	if err != nil {
		// 获取设备配置失败，请检查连接包是否正确
		logrus.Error(err)
		conn.Close()
		return
	}
	logrus.Info("获取设备配置成功：", tpGatewayConfig)
	// 将平台网关的配置存入全局变量
	globaldata.GateWayConfigMap.Store(regPkg, &tpGatewayConfig.Data)
	// 将设备连接存入全局变量
	globaldata.DeviceConnectionMap.Store(regPkg, &conn)
	m := *MQTT.MqttClient
	err = m.SendStatus(regPkg, "1")
	if err != nil {
		logrus.Info("SendStatus() failed, err: ", err)
	}
	// 设备上线
	logrus.Info("设备上线：", regPkg)
	HandleConn(regPkg) // 处理连接
	// defer conn.Close()
}
