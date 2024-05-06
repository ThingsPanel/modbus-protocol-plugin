# 开发帮助

## SDK升级
go get -u github.com/ThingsPanel/tp-protocol-sdk-go@v1.1.0
go get -u github.com/ThingsPanel/tp-protocol-sdk-go@v1.1.2
go get -u github.com/ThingsPanel/tp-protocol-sdk-go@v1.1.3
go get -u github.com/ThingsPanel/tp-protocol-sdk-go@v1.1.4
go get -u github.com/ThingsPanel/tp-protocol-sdk-go@v1.1.5
go get -u github.com/ThingsPanel/tp-protocol-sdk-go@v1.1.6
go get -u github.com/ThingsPanel/tp-protocol-sdk-go@v1.1.7
mosquitto_pub -h 47.115.210.16 -p 1883 -t "devices/telemetry" -m "{\"temp\":12.5}" -u "c55d8498" -P "c55d8498-e01e" -i "0"
mosquitto_pub -h 47.115.210.16 -p 1883 -t "devices/telemetry" -m "{\"temp\":12.5}" -u "c55d8498" -P "c55d8498-e01e" -i "0"

## 测试
设备ID：7fa6bf8d-4803-d1a3-2c0c-84d1cee4b9ba
pkg：xxxxxx

网关设备配置：MODBUS-TCP协议网关
子设备设备配置：MODBUS-TCP子设备配置

```json
{
	"id": "7fa6bf8d-4803-d1a3-2c0c-84d1cee4b9ba",
	"voucher": "{\"reg_pkg\":\"xxxxxx\"}",
	"device_type": "2",
	"protocol_type": "MODBUS_TCP",
	"config": {},
	"protocol_config_template": null,
	"sub_device": [
		{
			"device_id": "046832de-4f6d-7708-4a87-1a429c7dd580",
			"voucher": "{\"default\":\"47cbe486-6565-4ec9-6020-7579820636e5\"}",
			"sub_device_addr": "xxxxxx",
			"config": {},
			"protocol_config_template": {
				"CommandRawList": [
					{
						"DataIdentifierListStr": "ewqe",
						"DataType": "coil",
						"DecimalPlacesListStr": "ewq",
						"Endianess": "LITTLE",
						"EquationListStr": "ewq",
						"FunctionCode": 1,
						"Interval": "ewq",
						"Quantity": "ewq",
						"StartingAddress": "ewq"
					},
					{
						"DataIdentifierListStr": "fdsa",
						"DataType": "uint16",
						"DecimalPlacesListStr": "fdsa",
						"Endianess": "LITTLE",
						"EquationListStr": "fdsa",
						"FunctionCode": 2,
						"Interval": "fdsa",
						"Quantity": "fdsa",
						"StartingAddress": "fds"
					}
				],
				"SlaveID": "wqewq"
			}
		}
	]
}
```