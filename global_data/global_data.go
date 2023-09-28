package globaldata

import (
	"errors"
	"net"
	"sync"

	"github.com/ThingsPanel/tp-protocol-sdk-go/api"
)

// 平台网关配置map, key是网关的token，value是网关的配置
var GateWayConfigMap = make(map[string]*api.DeviceConfigResponseData)

// 设备连接map, key是设备的token，value是设备的连接
var DeviceConnectionMap = make(map[string]*net.Conn)

// 网关互斥锁map, key是网关的token，value是网关的互斥锁
var GateWayMutexMap = make(map[string]*sync.Mutex)

// 获取锁
func GetMutex(token string) error {
	if mutex, exists := GateWayMutexMap[token]; exists {
		mutex.Lock()
		return nil
	} else {
		return errors.New("锁不存在")
	}
}

// 释放锁
func ReleaseMutex(token string) error {
	if mutex, exists := GateWayMutexMap[token]; exists {
		mutex.Unlock()
		return nil
	} else {
		return errors.New("锁不存在")
	}
}
