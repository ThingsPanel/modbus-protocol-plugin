package services

import (
	"net"
	"strings"
	"sync"

	httpclient "github.com/ThingsPanel/modbus-protocol-plugin/http_client"
	"github.com/sirupsen/logrus"

	globaldata "github.com/ThingsPanel/modbus-protocol-plugin/global_data"
	MQTT "github.com/ThingsPanel/modbus-protocol-plugin/mqtt"
	"github.com/spf13/viper"
)

// 定义全局的conn管道
var connChan = make(chan net.Conn)

// 简单的IP限制
var (
	blockedIPs = make(map[string]bool)
	ipMutex    = &sync.Mutex{}
)

// 全局认证限流器
var authLimiter *AuthLimiter

func Start() {
	// 初始化认证限流器
	authLimiter = NewAuthLimiter()
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

		// 检查IP是否被限制
		clientIP := strings.Split(conn.RemoteAddr().String(), ":")[0]
		ipMutex.Lock()
		blocked := blockedIPs[clientIP]
		ipMutex.Unlock()

		if blocked {
			conn.Close()
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

func CloseConnection(conn net.Conn, regPkg string) {
	err := conn.Close()
	if err != nil {
		logrus.Info("Close() failed, err: ", err)
	}
	// 删除全局变量
	if m, exists := globaldata.DeviceConnectionMap.Load(regPkg); !exists {
		return
	} else if conn != *m.(*net.Conn) {
		return
	}
	logrus.Info("删除全局变量完成：", regPkg)
	// 做其他事情，比如发送离线消息
	m := *MQTT.MqttClient
	err = m.SendStatus(regPkg, "0")
	if err != nil {
		logrus.Info("SendStatus() failed, err: ", err)
	}
	globaldata.GateWayConfigMap.Delete(regPkg)
	globaldata.DeviceConnectionMap.Delete(regPkg)
	delete(globaldata.DeviceRWLock, regPkg)
	// 设备离线
	logrus.Info("设备离线：", regPkg)
}

// 验证连接并继续处理数据
func verifyConnection(conn net.Conn) {
	clientIP := strings.Split(conn.RemoteAddr().String(), ":")[0]

	// 检查IP是否被认证限流
	if authLimiter.IsBlocked(clientIP) {
		// 限制日志输出频率：每分钟最多1条
		if authLimiter.ShouldLogBlock(clientIP) {
			logrus.Warnf("IP认证被限流，连接已拒绝: %s", clientIP)
		}
		conn.Close()
		return
	}

	// 读取客户端发送的数据
	var buf [1024]byte
	n, err := conn.Read(buf[:])
	if err != nil {
		// 如果是连接重置错误，将IP加入黑名单
		if strings.Contains(err.Error(), "connection reset by peer") {
			ipMutex.Lock()
			blockedIPs[clientIP] = true
			ipMutex.Unlock()
			logrus.Info("IP已被限制: ", clientIP)
		} else {
			logrus.Info("Read() failed, err: ", err)
		}
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
		// 认证失败，记录限流
		authLimiter.RecordFailure(clientIP)
		// 获取设备配置失败，请检查连接包是否正确
		logrus.Error(err)
		conn.Close()
		return
	}

	// 认证成功，清除限流记录
	authLimiter.RecordSuccess(clientIP)

	logrus.Info("获取设备配置成功：", tpGatewayConfig)
	// 将平台网关的配置存入全局变量
	globaldata.GateWayConfigMap.Store(tpGatewayConfig.Data.ID, &tpGatewayConfig.Data)
	// 将设备连接存入全局变量
	globaldata.DeviceConnectionMap.Store(tpGatewayConfig.Data.ID, &conn)
	m := *MQTT.MqttClient
	err = m.SendStatus(tpGatewayConfig.Data.ID, "1")
	if err != nil {
		logrus.Info("SendStatus() failed, err: ", err)
	}
	// 设备上线
	logrus.Info("设备上线(", tpGatewayConfig.Data.ID, "):", regPkg)
	HandleConn(regPkg, tpGatewayConfig.Data.ID) // 处理连接
	// defer conn.Close()
}
