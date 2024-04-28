# 开发帮助

## SDK升级
go get -u github.com/ThingsPanel/tp-protocol-sdk-go@v1.1.0
go get -u github.com/ThingsPanel/tp-protocol-sdk-go@v1.1.2
go get -u github.com/ThingsPanel/tp-protocol-sdk-go@v1.1.3
mosquitto_pub -h 47.115.210.16 -p 1883 -t "devices/telemetry" -m "{\"temp\":12.5}" -u "c55d8498" -P "c55d8498-e01e" -i "0"

mosquitto_pub -h 47.115.210.16 -p 1883 -t "devices/telemetry" -m "{\"temp\":12.5}" -u "c55d8498" -P "c55d8498-e01e" -i "0"