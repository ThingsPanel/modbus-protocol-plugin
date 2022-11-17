package mqtt

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	server_map "tp-modbus/map"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/go-basic/uuid"
	"github.com/panjf2000/ants/v2"
	"github.com/spf13/viper"
	"github.com/tbrandon/mbserver"
)

var mqtt_client mqtt.Client

func init() {
	listenMQTT()
}
func listenMQTT() {
	broker := os.Getenv("MQTT_HOST")
	if broker == "" {
		broker = viper.GetString("mqtt.broker")
	}
	clientid := uuid.New()
	username := viper.GetString("mqtt.username")
	password := viper.GetString("mqtt.password")

	var connectLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
		fmt.Printf("Connect lost: %v", err)
	}
	opts := mqtt.NewClientOptions()
	opts.SetUsername(username)
	opts.SetPassword(password)
	opts.SetClientID(clientid)
	opts.AddBroker(broker)
	opts.SetAutoReconnect(true) //自动重连
	opts.SetOrderMatters(false)
	opts.OnConnectionLost = connectLostHandler
	opts.SetOnConnectHandler(func(c mqtt.Client) {
		log.Println("MQTT客户端连接成功...", broker)
	})
	p, _ := ants.NewPool(viper.GetInt("mqtt.pool")) //设置并发池
	log.Println("mqtt客户端订阅处理的并发池大小为", viper.GetInt("mqtt.subscribe_pool"))
	opts.SetDefaultPublishHandler(func(c mqtt.Client, m mqtt.Message) {
		_ = p.Submit(func() {
			MsgProc(c, m)
		})
	})
	mqtt_client = mqtt.NewClient(opts)
	if token := mqtt_client.Connect(); token.Wait() && token.Error() != nil {
		log.Println("mqtt客户端连接异常...", viper.GetString("mqtt.broker"), token.Error())
		os.Exit(1)
	}
	if token := mqtt_client.Subscribe(viper.GetString("mqtt.topic_to_subscribe"), 0, nil); token.Wait() && token.Error() != nil {
		log.Println("mqtt订阅异常异常...", viper.GetString("mqtt.topic_to_subscribe"), token.Error())
		os.Exit(1)
	} else {
		log.Println("mqtt订阅成功...", viper.GetString("mqtt.topic_to_subscribe"))
	}
}

// 接收订阅的消息进行处理
func MsgProc(c mqtt.Client, m mqtt.Message) {
	log.Println("收到订阅消息", string(m.Payload()))
	// plugin/modbus/#
	// 获取子设备id
	d := strings.Split(m.Topic(), "/")
	sub_device_id := d[len(d)-1]
	sub_device_config := server_map.SubDeviceConfigMap[sub_device_id]
	log.Println("子设备配置：", server_map.SubDeviceConfigMap[sub_device_id])
	// 根据子设备的配置和mqtt消息中的属性确定每个属性的起始地址
	pt := server_map.GatewayConfigMap[sub_device_config.GatewayId].ProtocolType
	if pt == "MODBUS_RTU" {
		var frame mbserver.RTUFrame
		var starting_address uint16
		var address_num uint16
		frame.Address = server_map.SubDeviceConfigMap[sub_device_id].DeviceAddress // 设备地址
		log.Println("功能码：", server_map.SubDeviceConfigMap[sub_device_id].FunctionCode)
		switch server_map.SubDeviceConfigMap[sub_device_id].FunctionCode {
		case uint8(1): // 写线圈
			frame.Function = 5
			// 找出消息的key对应的位置并组长字节报文
			var msg map[string]interface{}
			json.Unmarshal(m.Payload(), &msg)
			key_list := strings.Split(server_map.SubDeviceConfigMap[sub_device_id].Key, ",")
			for key, value := range msg {
				var number = 0
				for i := 0; i < len(key_list); i++ {
					if key == key_list[i] {
						number = i
						break
					}
				}
				starting_address = server_map.SubDeviceConfigMap[sub_device_id].StartingAddress + uint16(number)
				// FF00为ON，0000为OFF
				n := uint8(value.(float64)) * 255
				address_num = 1
				fmt.Println(starting_address, address_num)
				addr_b := make([]byte, 2)
				binary.BigEndian.PutUint16(addr_b, starting_address)
				// 设备地址|功能码|{线圈地址|数据}|校验码
				var data_b = []byte{addr_b[0], addr_b[1], n, 0}
				frame.SetData(data_b)
				SendMessage(&frame, sub_device_config.GatewayId, sub_device_config.DeviceId, frame.Bytes()) //发送指令给网关设备
				init_frame := server_map.RTUFrameMap[sub_device_id]
				SendMessage(&init_frame, sub_device_config.GatewayId, sub_device_config.DeviceId, init_frame.Bytes())
			}
		case uint8(3): //写寄存器
			frame.Function = 6
		}
		//mbserver.SetDataWithRegisterAndNumber(&frame, server_map.SubDeviceConfigMap[deviceId].StartingAddress, server_map.SubDeviceConfigMap[deviceId].AddressNum)
		//mqtt.SendMessage(&frame, gatewayId, deviceId, frame.Bytes()) //发送指令给网关设备
	}

	// 组装字节报文发送给设备

}

//发送消息
func Send(payload []byte) (err error) {
	t := mqtt_client.Publish(viper.GetString("mqtt.topic_to_publish"), 1, false, string(payload))
	if t.Error() != nil {
		log.Println("发送消息失败...", string(payload), t.Error())
	} else {
		log.Println("发送...", string(payload))
	}
	return t.Error()
}
