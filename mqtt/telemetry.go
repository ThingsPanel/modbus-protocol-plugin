package mqtt

import (
	"encoding/json"
	"fmt"

	"github.com/ThingsPanel/tp-protocol-sdk-go/api"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

const (
	gatewayDeviceType     = "2"
	gatewayTelemetryTopic = "gateway/telemetry"
)

// PublishCollectedTelemetry routes collected values by the platform's device type.
func PublishCollectedTelemetry(device *api.DeviceConfigResponseData, subDevice *api.SubDevice, data map[string]interface{}) error {
	if device.DeviceType != gatewayDeviceType {
		return PublishDeviceTelemetry(subDevice.DeviceID, data)
	}
	if device.ID == "" || subDevice.SubDeviceAddr == "" {
		return fmt.Errorf("gateway ID and sub-device address are required for gateway telemetry")
	}

	values := map[string]interface{}{
		"sub_device_data": map[string]map[string]interface{}{
			subDevice.SubDeviceAddr: data,
		},
	}
	return publishTelemetry(gatewayTelemetryTopic, device.ID, values)
}

func PublishDeviceTelemetry(deviceID string, data map[string]interface{}) error {
	return publishTelemetry(viper.GetString("mqtt.topic_to_publish_sub"), deviceID, data)
}

func publishTelemetry(topic, deviceID string, data map[string]interface{}) error {
	values, err := json.Marshal(data)
	if err != nil {
		return err
	}
	logrus.Info("values:", string(values))

	payload, err := json.Marshal(map[string]interface{}{
		"device_id": deviceID,
		"values":    values,
	})
	if err != nil {
		return err
	}
	return publish(topic, string(payload))
}
