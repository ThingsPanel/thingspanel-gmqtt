# syntax=docker/dockerfile:1
FROM swr.cn-east-2.myhuaweicloud.com/library/golang:latest AS builder
WORKDIR $GOPATH/src/app
ADD . ./
ENV GO111MODULE=on
ENV GOPROXY="https://goproxy.io"
# 确保我们处于项目根目录
RUN go mod init github.com/DrmagicE/gmqtt || true
RUN go mod tidy
# 明确获取 crypto 包
RUN go get golang.org/x/crypto/ed25519
WORKDIR $GOPATH/src/app/cmd/gmqttd
RUN go build -v

FROM swr.cn-east-2.myhuaweicloud.com/library/busybox:latest
WORKDIR /gmqttd
COPY --from=builder /go/src/app/cmd/gmqttd/gmqttd .
# 确保配置文件存在
COPY --from=builder /go/src/app/cmd/gmqttd/default_config.yml /gmqttd/ || true
EXPOSE 1883 8883 8082 8083 8084
RUN chmod +x gmqttd
RUN pwd
RUN ls -lrt
ENTRYPOINT ["./gmqttd", "start", "-c", "/gmqttd/default_config.yml"]