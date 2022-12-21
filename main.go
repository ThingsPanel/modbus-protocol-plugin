package main

import (
	"bufio"
	"log"
	"net"
	"sync"
	"time"
	server_map "tp-modbus/map"
	_ "tp-modbus/src/initialize"
	"tp-modbus/src/mqtt"
	"tp-modbus/src/tp"

	"github.com/spf13/viper"
)

// 连接处理
func linkProcess(conn net.Conn) {
	reader := bufio.NewReader(conn)
	var buf [128]byte
	n, err := reader.Read(buf[:]) // 读取数据
	if err != nil {
		log.Println("read from client failed, err: ", err)
		return
	}
	accessToken := string(buf[:n])
	log.Println("收到网关设备发来的密钥：", accessToken)
	time.Sleep(time.Second * 1) // 建立连接后暂停一秒（有些设备需要等待）
	if string(buf[:n]) != "" {
		gatewayConfig, err := tp.GetGatewayConfig(accessToken) // 校验密钥并获取网关设备配置
		if err != nil {
			log.Println("密钥验证失败...", accessToken, err.Error())
			return
		} else {
			//设备上线
			mqtt.SendStatus(accessToken, "1")
		}
		server_map.TcpClientMap[gatewayConfig.Id] = conn                   // 在集合中添加tcp连接
		delete(server_map.GatewayChannelMap, gatewayConfig.Id)             // 删除原网关设备通道并新建通道
		server_map.GatewayChannelMap[gatewayConfig.Id] = make(chan int, 1) // 创建网关通道
		var s sync.Mutex
		server_map.TcpClientSyncMap[gatewayConfig.Id] = &s
		//go process(conn, gatewayConfig.GatewayId)
		tp.ProcessReq(gatewayConfig.Id) // 启动一个goroutine来处理客户端的连接请求
	}
}

func main() {
	listen, err := net.Listen("tcp", viper.GetString("server.address"))
	if err != nil {
		log.Println("Listen() failed, err: ", err)
		return
	}
	log.Println("服务启动成功：", viper.GetString("server.address"))
	for {
		conn, err := listen.Accept() // 监听客户端的连接请求
		if err != nil {
			log.Println("Accept() failed, err: ", err)
			continue
		}
		linkProcess(conn)
	}
}
