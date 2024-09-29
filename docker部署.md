# docker部署

## 构建镜像

```bash
docker build -t modbus-protocol-plugin:latest .
```

## 如果平台是使用产品提供的docker-compose 部署的话，那么只需要在docker-compose.yml文件中添加以下内容即可：

```yml
  modbus_service:
    image: modbus-protocol-plugin:latest  # 使用我们刚刚构建的镜像
    ports:
      - "502:502"
      - "503:503"
    environment:
      - "MODBUS_THINGSPANEL_ADDRESS=http://172.50.0.2:9999" # 这里的地址是物联网平台的地址
      - "MODBUS_MQTT_BROKER=172.50.0.5:1883" # 这里的地址是mqtt的地址
      - "MODBUS_MQTT_QOS=0"
    networks:
      extnetwork:
        ipv4_address: 172.50.0.7
    depends_on:
      - backend
      - gmqtt
    restart: unless-stopped
```

- 注意：如果您修改了 Docker Compose 文件，可能需要先运行以下命令(这会重新创建 modbus_service 容器而不重启其依赖的服务)：

```bash
docker-compose up -d --no-deps modbus_service
```

- 如果需要查看日志，可以运行：

```bash
docker-compose logs -f modbus_service
```
