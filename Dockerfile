# syntax=docker/dockerfile:1
FROM swr.cn-east-2.myhuaweicloud.com/library/golang:1.17 AS builder
WORKDIR /go/src/app
COPY . .
ENV GO111MODULE=on
ENV GOPROXY="https://goproxy.io"
WORKDIR /go/src/app/cmd/gmqttd
RUN go build -v

FROM swr.cn-east-2.myhuaweicloud.com/library/busybox:latest
WORKDIR /gmqttd
COPY --from=builder /go/src/app/cmd/gmqttd/gmqttd .
COPY --from=builder /go/src/app/cmd/gmqttd/default_config.yml /gmqttd/ || true
EXPOSE 1883 8883 8082 8083 8084
RUN chmod +x gmqttd
ENTRYPOINT ["./gmqttd", "start", "-c", "/gmqttd/default_config.yml"]