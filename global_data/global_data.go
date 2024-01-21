package globaldata

import (
	"sync"
)

// 平台网关配置map, key是网关的token，value是网关的配置
//var GateWayConfigMap = make(map[string]*api.DeviceConfigResponseData)
var GateWayConfigMap sync.Map

// 设备连接map, key是设备的token，value是设备的连接
//var DeviceConnectionMap = make(map[string]*net.Conn)
var DeviceConnectionMap sync.Map
