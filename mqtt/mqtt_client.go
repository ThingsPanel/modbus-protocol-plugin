package mqtt

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	globaldata "github.com/ThingsPanel/modbus-protocol-plugin/global_data"
	"github.com/ThingsPanel/modbus-protocol-plugin/modbus"
	tpconfig "github.com/ThingsPanel/modbus-protocol-plugin/tp_config"
	"github.com/ThingsPanel/tp-protocol-sdk-go/api"

	tpprotocolsdkgo "github.com/ThingsPanel/tp-protocol-sdk-go"
	MQTT "github.com/eclipse/paho.mqtt.golang"
	"github.com/spf13/viper"
)

var MqttClient *tpprotocolsdkgo.MQTTClient

func InitClient() {
	log.Println("创建mqtt客户端")
	// 创建新的MQTT客户端实例
	addr := viper.GetString("mqtt.broker")
	username := viper.GetString("mqtt.username")
	password := viper.GetString("mqtt.password")
	client := tpprotocolsdkgo.NewMQTTClient(addr, username, password)
	// 尝试连接到MQTT代理
	if err := client.Connect(); err != nil {
		log.Fatalf("连接失败: %v", err)
	}
	log.Println("连接成功")
	MqttClient = client
}

// 发布设备消息{"token":device_token,"values":{sub_device_addr1:{key:value...},sub_device_add2r:{key:value...}}}
func Publish(payload string) error {
	// 主题
	topic := viper.GetString("mqtt.topic_to_publish")
	qos := viper.GetUint("mqtt.qos")
	// 发布消息
	if err := MqttClient.Publish(topic, string(payload), uint8(qos)); err != nil {
		log.Printf("发布消息失败: %v", err)
		return err
	}
	log.Println("发布消息成功:", payload, "主题:", topic)
	return nil
}

// 订阅
func Subscribe() {
	// 主题
	topic := viper.GetString("mqtt.topic_to_subscribe")
	qos := viper.GetUint("mqtt.qos")
	// 订阅主题
	if err := MqttClient.Subscribe(topic, messageHandler, uint8(qos)); err != nil {
		log.Printf("订阅主题失败: %v", err)
	}
	log.Println("订阅主题成功:", topic)

}

// 设备下发消息的回调函数：主题plugin/modbus/# payload：{sub_device_addr:{key:value...},sub_device_addr:{key:value...}}
func messageHandler(client MQTT.Client, msg MQTT.Message) {
	fmt.Printf("Received message on topic: %s\nMessage: %s\n", msg.Topic(), msg.Payload())
	// 解析主题获取token（plugin/modbus/# #为token）
	token := msg.Topic()[14:]
	// 解析payload的json报文
	payloadMap := make(map[string]interface{})
	if err := json.Unmarshal(msg.Payload(), &payloadMap); err != nil {
		log.Println(err)
		return
	}
	// 遍历payloadMap，获取sub_device_addr和key-value
	for subDeviceAddr, dataMap := range payloadMap {
		var gateWayConfigMap *api.DeviceConfigResponseData
		// 获取网关配置
		log.Println("token:", token)
		if m, exists := globaldata.GateWayConfigMap.Load(token); !exists {
			log.Println("网关没有连接")
			return
		} else {
			gateWayConfigMap = m.(*api.DeviceConfigResponseData)
		}
		// 遍历subDevices
		for _, subDevice := range gateWayConfigMap.SubDevices {
			if subDevice.SubDeviceAddr == subDeviceAddr {
				// 遍历配置项
				subDeviceFormConfig, err := tpconfig.NewSubDeviceFormConfig(subDevice.Config)
				if err != nil {
					log.Println(err)
					continue
				}
				// 首先遍历dataMap
				for key, value := range dataMap.(map[string]interface{}) {
					// 遍历配置项
					for _, commandRaw := range subDeviceFormConfig.CommandRawList {
						// 遍历配置项的key
						for i, configKey := range strings.Split(commandRaw.DataIdetifierListStr, ",") {
							if key == strings.TrimSpace(configKey) {
								// 根据配置项的数据类型，将value转为对应的数据类型
								functionCode, startAddress, data, err := commandRaw.GetWriteCommand(key, value, i)
								if err != nil {
									log.Println(err)
									continue
								}
								if gateWayConfigMap.ProtocolType == "MODBUS_RTU" {
									// 创建RTUCommand
									RTUCommand := modbus.NewRTUCommand(subDeviceFormConfig.SlaveID, functionCode, startAddress, 1, modbus.EndianessType(commandRaw.Endianess))
									RTUCommand.ValueData = data

									sendData, err := RTUCommand.Serialize()
									if err != nil {
										log.Println(err)
										return
									}
									err = handleDeviceConnection(token, sendData)
									if err != nil {
										log.Println(err)
										return
									}
									// 反序列化数据
								} else if m, ok := globaldata.GateWayConfigMap.Load(token); ok && m.(*api.DeviceConfigResponseData).ProtocolType == "MODBUS_TCP" {
									// 创建TCPCommand
									TCPCommand := modbus.NewTCPCommand(subDeviceFormConfig.SlaveID, functionCode, startAddress, 1, modbus.EndianessType(commandRaw.Endianess))
									TCPCommand.ValueData = data
									sendData, err := TCPCommand.Serialize()
									if err != nil {
										log.Println(err)
										return
									}
									err = handleDeviceConnection(token, sendData)
									if err != nil {
										log.Println(err)
										return
									}
								}
							}
						}
					}
				}
			}
		}
	}
}

// 处理设备连接
func handleDeviceConnection(token string, sendData []byte) error {
	// 获取连接
	c, exists := globaldata.DeviceConnectionMap.Load(token)
	if !exists {
		return fmt.Errorf("网关没有连接")
	}
	conn := *c.(*net.Conn)

	// 设置写超时时间
	err := conn.SetWriteDeadline(time.Now().Add(15 * time.Second))
	if err != nil {
		log.Println("SetWriteDeadline() failed, err: ", err)
		return err
	}
	log.Println("AccessToken:", token, "控制设备请求：", sendData)
	_, err = conn.Write(sendData)
	if err != nil {
		return fmt.Errorf("写入失败: %v", err)
	}

	// 读取数据
	// 设置读取超时时间
	err = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	if err != nil {
		log.Println("SetReadDeadline() failed, err: ", err)
		return err
	}
	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil {
		return fmt.Errorf("读取失败: %v", err)
	}

	log.Println("AccessToken:", token, "控制设备响应：", buf[:n])
	return nil
}
