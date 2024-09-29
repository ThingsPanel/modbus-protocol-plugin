# syntax=docker/dockerfile:1
FROM golang:alpine
WORKDIR $GOPATH/src/app
ADD . ./
ENV GO111MODULE=on
ENV GOPROXY="https://goproxy.io"
ENV MODBUS_THINGSPANEL_ADDRESS=http://127.0.0.1:9999
ENV MODBUS_MQTT_BROKER=127.0.0.1:1883
ENV MODBUS_MQTT_QOS=0
RUN go build
EXPOSE 502
EXPOSE 503
RUN chmod +x modbus-protocol-plugin
RUN pwd
RUN ls -lrt
ENTRYPOINT [ "./modbus-protocol-plugin" ]