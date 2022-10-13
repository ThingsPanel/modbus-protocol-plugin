package modbus_rtu

import (
	"testing"
	"time"
	server_map "tp-modbus/map"
)

// Function 2
func TestInitTCPGo(t *testing.T) {
	go InitTCPGo("001", "002")
	time.Sleep(6 * time.Second)
	server_map.GatewayChannelMap["001"] <- 1
	close(server_map.GatewayChannelMap["001"])
	time.Sleep(2 * time.Second)
}
