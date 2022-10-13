package mqtt

import (
	"fmt"
	"log"
	"os"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/go-basic/uuid"
	"github.com/panjf2000/ants/v2"
	"github.com/spf13/viper"
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
			MsgProc(m)
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
func MsgProc(m mqtt.Message) {
	log.Println("收到订阅消息", string(m.Payload()))
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
