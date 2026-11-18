package mqtt

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/ThingsPanel/tp-protocol-sdk-go/api"
	MQTT "github.com/eclipse/paho.mqtt.golang"
	"github.com/sirupsen/logrus"
)

const commandPrefix = "plugin/modbus/devices/telemetry/control/"

type managedBroker struct {
	client   MQTT.Client
	mu       sync.Mutex
	gateways map[string][]string
}

func connectManagedBroker(address, username, password string) (*managedBroker, error) {
	if address == "" || username == "" || password == "" {
		return nil, errors.New("managed MQTT address and credentials are required")
	}
	b := &managedBroker{gateways: make(map[string][]string)}
	options := MQTT.NewClientOptions().AddBroker(address).SetClientID("modbus-managed").
		SetUsername(username).SetPassword(password).SetConnectTimeout(5 * time.Second).
		SetOrderMatters(false).
		SetConnectRetry(true).SetConnectRetryInterval(2 * time.Second).SetAutoReconnect(true).
		SetOnConnectHandler(func(client MQTT.Client) {
			b.mu.Lock()
			defer b.mu.Unlock()
			for _, topics := range b.gateways {
				b.subscribe(topics)
			}
		})
	b.client = MQTT.NewClient(options)
	if err := mqttResult(b.client.Connect(), 75*time.Second); err != nil {
		return nil, fmt.Errorf("managed MQTT connection: %w", err)
	}
	return b, nil
}

func mqttResult(token MQTT.Token, timeout time.Duration) error {
	if !token.WaitTimeout(timeout) {
		return errors.New("MQTT operation timed out")
	}
	return token.Error()
}

func (b *managedBroker) Publish(topic, payload string, qos uint8) error {
	return mqttResult(b.client.Publish(topic, qos, false, payload), 5*time.Second)
}

func (b *managedBroker) SendStatus(deviceID, status string) error {
	return b.Publish("devices/status/"+deviceID, status, 1)
}

func (b *managedBroker) subscribe(topics []string) {
	filters := make(map[string]byte, len(topics))
	for _, topic := range topics {
		filters[topic] = 1
	}
	if len(filters) > 0 {
		if err := mqttResult(b.client.SubscribeMultiple(filters, messageHandler), 5*time.Second); err != nil {
			logrus.WithError(err).Warn("managed Modbus control subscription failed")
		}
	}
}

func SetGatewaySubscriptions(gatewayID string, devices []api.SubDevice) {
	b, ok := MqttClient.(*managedBroker)
	if !ok {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if previous := b.gateways[gatewayID]; len(previous) > 0 && b.client.IsConnectionOpen() {
		if err := mqttResult(b.client.Unsubscribe(previous...), 5*time.Second); err != nil {
			logrus.WithError(err).Warn("managed Modbus control unsubscribe failed")
		}
	}
	delete(b.gateways, gatewayID)
	var topics []string
	for _, device := range devices {
		if device.DeviceID != "" && !strings.ContainsAny(device.DeviceID, "/+#\x00") {
			topics = append(topics, commandPrefix+device.DeviceID)
		}
	}
	if len(topics) > 0 {
		b.gateways[gatewayID] = topics
		if b.client.IsConnectionOpen() {
			b.subscribe(topics)
		}
	}
}

func Ready() bool {
	b, ok := MqttClient.(*managedBroker)
	return MqttClient != nil && (!ok || b.client.IsConnectionOpen())
}
