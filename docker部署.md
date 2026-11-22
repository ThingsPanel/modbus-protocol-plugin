# docker部署

## 构建镜像

```bash
docker build -t modbus-protocol-plugin:latest .
```

## 使用 Docker Compose 部署

插件通过标准 MQTT 协议连接平台配置的 broker，不要求使用某个特定 broker 产品。先设置可从插件容器访问的 broker 地址和账号：

```bash
export MODBUS_MQTT_BROKER='tcp://mqtt-broker:1883'
export MODBUS_MQTT_USERNAME='your-mqtt-username'
export MODBUS_MQTT_PASSWORD='your-mqtt-password'
```

确保 MQTT 账号有权发布 `devices/telemetry` 和 `devices/status/#`，并订阅 `plugin/modbus/#`。将 broker 主机名替换为当前环境中的地址；若 broker 在另一个 Docker 网络，使用可路由的地址并开放相应端口。

在 `docker-compose.yml` 中添加：

```yml
  modbus_service:
    image: modbus-protocol-plugin:latest
    ports:
      - "502:1502"
      - "503:1503"
    environment:
      MODBUS_THINGSPANEL_ADDRESS: "http://backend:9999"
      MODBUS_MQTT_BROKER: "${MODBUS_MQTT_BROKER:?set MODBUS_MQTT_BROKER}"
      MODBUS_MQTT_USERNAME: "${MODBUS_MQTT_USERNAME:?set MODBUS_MQTT_USERNAME}"
      MODBUS_MQTT_PASSWORD: "${MODBUS_MQTT_PASSWORD:?set MODBUS_MQTT_PASSWORD}"
      MODBUS_MQTT_QOS: "0"
    networks:
      - thingspanel_network
    depends_on:
      - backend
    restart: unless-stopped
```

The broker does not need to be a Compose dependency of the plugin. Configure the broker endpoint and credentials for each deployment; the plugin keeps the ThingsPanel telemetry topic and payload contract.

- 注意：如果您修改了 Docker Compose 文件，可能需要先运行以下命令(这会重新创建 modbus_service 容器而不重启其依赖的服务)：

```bash
docker-compose up -d --no-deps modbus_service
```

- 如果需要查看日志，可以运行：

```bash
docker-compose logs -f modbus_service
```
