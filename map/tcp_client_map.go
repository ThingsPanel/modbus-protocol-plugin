package server_map

import (
	"net"
	"sync"
)

var TcpClientMap = make(map[string]net.Conn)        // 网关tcp客户端集合
var TcpClientSyncMap = make(map[string]*sync.Mutex) // 网关tcp客户端互斥锁（保证在处理反馈的时候不受干扰）
