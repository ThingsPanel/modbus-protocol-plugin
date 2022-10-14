# syntax=docker/dockerfile:1
FROM golang:alpine
WORKDIR $GOPATH/src/app
ADD . ./
ENV GO111MODULE=on
ENV GOPROXY="https://goproxy.io"
ENV TP_HOST=172.19.0.2:9999
ENV MQTT_HOST=172.19.0.5:1883
RUN go build
EXPOSE 502
EXPOSE 503
RUN chmod +x tp-modbus
RUN pwd
RUN ls -lrt
ENTRYPOINT [ "./tp-modbus" ]