package mqtt

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/ThingsPanel/tp-protocol-sdk-go/api"
	"github.com/spf13/viper"
)

type telemetryRecorder struct {
	topic   string
	payload string
}

func (r *telemetryRecorder) Publish(topic, payload string, _ uint8) error {
	r.topic, r.payload = topic, payload
	return nil
}

func (r *telemetryRecorder) SendStatus(_, _ string) error {
	return nil
}

func TestTelemetryRouting(t *testing.T) {
	previousClient := MqttClient
	previousTopic := viper.GetString("mqtt.topic_to_publish_sub")
	t.Cleanup(func() {
		MqttClient = previousClient
		viper.Set("mqtt.topic_to_publish_sub", previousTopic)
	})
	viper.Set("mqtt.topic_to_publish_sub", "devices/telemetry")

	subDevice := &api.SubDevice{DeviceID: "child-id", SubDeviceAddr: "01"}
	data := map[string]interface{}{"weight": 3.9}
	gatewayValues := map[string]interface{}{
		"sub_device_data": map[string]interface{}{"01": data},
	}
	for _, tc := range []struct {
		name      string
		publish   func() error
		topic     string
		deviceID  string
		values    map[string]interface{}
		wantError bool
	}{
		{
			name: "gateway selected by device type",
			publish: func() error {
				return PublishCollectedTelemetry(&api.DeviceConfigResponseData{ID: "gateway-id", DeviceType: "2", ProtocolType: "OTHER"}, subDevice, data)
			},
			topic: "gateway/telemetry", deviceID: "gateway-id", values: gatewayValues,
		},
		{
			name: "non-gateway preserves child telemetry",
			publish: func() error {
				return PublishCollectedTelemetry(&api.DeviceConfigResponseData{ID: "device-id", DeviceType: "1", ProtocolType: "MODBUS_TCP"}, subDevice, data)
			},
			topic: "devices/telemetry", deviceID: "child-id", values: data,
		},
		{
			name: "control response preserves child telemetry",
			publish: func() error {
				return PublishRsponse("weight", 3.9, subDevice.DeviceID)
			},
			topic: "devices/telemetry", deviceID: "child-id", values: data,
		},
		{
			name: "gateway telemetry requires platform child address",
			publish: func() error {
				return PublishCollectedTelemetry(&api.DeviceConfigResponseData{ID: "gateway-id", DeviceType: "2"}, &api.SubDevice{DeviceID: "child-id"}, data)
			},
			wantError: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			recorder := &telemetryRecorder{}
			MqttClient = recorder
			err := tc.publish()
			if tc.wantError {
				if err == nil || recorder.topic != "" {
					t.Fatalf("expected validation error without publishing, got %v and topic %q", err, recorder.topic)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if recorder.topic != tc.topic {
				t.Fatalf("topic = %q, want %q", recorder.topic, tc.topic)
			}
			var envelope struct {
				DeviceID string `json:"device_id"`
				Values   []byte `json:"values"`
			}
			if err := json.Unmarshal([]byte(recorder.payload), &envelope); err != nil {
				t.Fatal(err)
			}
			if envelope.DeviceID != tc.deviceID {
				t.Fatalf("device_id = %q, want %q", envelope.DeviceID, tc.deviceID)
			}
			var values map[string]interface{}
			if err := json.Unmarshal(envelope.Values, &values); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(values, tc.values) {
				t.Fatalf("values = %#v, want %#v", values, tc.values)
			}
		})
	}
}
