# syntax=docker/dockerfile:1
FROM swr.cn-east-2.myhuaweicloud.com/library/golang:latest AS builder
WORKDIR $GOPATH/src/app
ADD . ./
ENV GO111MODULE on
ENV GOPROXY="https://goproxy.io"
WORKDIR $GOPATH/src/app/cmd/gmqttd
RUN go build

# 使用华为云的 busybox 镜像代替 alpine:3.12
FROM swr.cn-east-2.myhuaweicloud.com/library/busybox:latest
WORKDIR /gmqttd
# busybox 没有 apk，如果需要时区数据，需要使用不同的方式安装
# 或者可以从构建阶段复制时区数据
COPY --from=builder /go/src/app/cmd/gmqttd . 
EXPOSE 1883 8883 8082 8083 8084
RUN chmod +x gmqttd
RUN pwd
RUN ls -lrt
ENTRYPOINT ["./gmqttd", "start", "-c", "/gmqttd/default_config.yml"]