# TCP Keep-Alive 测试程序

## 功能说明

这个测试程序用于复现和观察 GMQTT 的 TCP Keep-Alive 问题（每 15 秒发送一次 TCP Keep-Alive ACK 包）。

## 使用步骤

### 1. 确保 GMQTT 服务器正在运行

```bash
cd /mnt/c/B-work/thingspanel-work/1-code/1-tp/thingspanel-gmqtt/cmd/gmqttd
go run . start -c thingspanel.yml
```

### 2. 在另一个终端启动抓包

#### 使用 tshark（Windows/Linux 推荐）

**方法 1: 基本抓包（显示所有包）**
```bash
# Windows PowerShell
tshark -i 8 -f "tcp port 1883" -nn -t ad

# Linux
tshark -i lo -f "tcp port 1883" -t ad
```

**方法 2: 只看 TCP Keep-Alive 相关的包（推荐）**
```bash
# Windows PowerShell
tshark -i 8 -f "tcp port 1883" -t ad -Y "tcp.flags.ack==1 && tcp.len==0"

# Linux
tshark -i lo -f "tcp port 1883" -t ad -Y "tcp.flags.ack==1 && tcp.len==0"
```

**方法 3: 显示详细信息**
```bash
# Windows PowerShell
tshark -i 8 -f "tcp port 1883" -t ad -V

# Linux
tshark -i lo -f "tcp port 1883" -t ad -V
```

**方法 4: 保存到文件供 Wireshark 分析**
```bash
# Windows PowerShell
tshark -i 8 -f "tcp port 1883" -w C:\temp\mqtt_keepalive.pcap

# Linux
tshark -i lo -f "tcp port 1883" -w /tmp/mqtt_keepalive.pcap
```

**参数说明**:
- `-i 1` 或 `-i lo`: 指定网络接口（Windows 通常是 1，Linux 本地回环是 lo）
- `-f "tcp port 1883"`: 捕获过滤器，只抓 1883 端口的 TCP 包
- `-t ad`: 时间戳格式（absolute with date，绝对时间+日期）
- `-Y`: 显示过滤器，用于过滤要显示的包
- `-V`: 显示详细信息
- `-w`: 写入文件

#### 使用 tcpdump（仅 Linux）

**方法 1: 使用 tcpdump（推荐）**
```bash
sudo tcpdump -i lo port 1883 -nn -tttt
```

**方法 2: 只看 ACK 包**
```bash
sudo tcpdump -i lo port 1883 -nn -tttt | grep ACK
```

**方法 3: 保存到文件供 Wireshark 分析**
```bash
sudo tcpdump -i lo port 1883 -w /tmp/mqtt_keepalive.pcap
```

### 3. 运行测试程序

```bash
cd /mnt/c/B-work/thingspanel-work/1-code/1-tp/thingspanel-gmqtt/cmd/keep_alive_test
go run main.go
```

## 预期观察结果

### 控制台输出
```
========================================
TCP Keep-Alive 问题复现测试
========================================
MQTT Broker: tcp://127.0.0.1:1883
用户名: root
客户端ID: tcp-keepalive-test-client
========================================
[配置] MQTT Keep-Alive: 60 秒
[连接中...] 正在连接到 MQTT Broker
[✓] MQTT 客户端已连接成功
========================================
现在开始监控 TCP Keep-Alive 行为
请使用 Wireshark 或 tcpdump 抓包观察:
  sudo tcpdump -i any port 1883 -nn -tttt
或者:
  sudo tcpdump -i lo port 1883 -nn -tttt | grep ACK
========================================
预期现象: 每隔 15 秒会看到 TCP Keep-Alive ACK 包
========================================
[订阅] 订阅主题: test/keepalive
[✓] 订阅成功
========================================
[运行中] 按 Ctrl+C 停止测试
========================================

提示:
1. 在另一个终端运行抓包命令观察 TCP Keep-Alive
2. 你应该能看到每隔 15 秒一个 [ACK] 包
3. 这就是 Go net 包默认的 TCP Keep-Alive 行为

[运行时间] 10s (继续观察 TCP Keep-Alive 包...)
[运行时间] 20s (继续观察 TCP Keep-Alive 包...)
[发送消息] Topic: test/keepalive, Payload: 测试消息 #1 - 时间: 14:23:45
[✓] 消息发送成功
[收到消息] Topic: test/keepalive, Payload: 测试消息 #1 - 时间: 14:23:45
```

### tcpdump 抓包输出示例

```
2025-10-29 14:23:15.123456 IP 127.0.0.1.54321 > 127.0.0.1.1883: Flags [S], seq 123456789
2025-10-29 14:23:15.123789 IP 127.0.0.1.1883 > 127.0.0.1.54321: Flags [S.], seq 987654321, ack 123456790
2025-10-29 14:23:15.124000 IP 127.0.0.1.54321 > 127.0.0.1.1883: Flags [.], ack 1

# ⬇️ 这里是 TCP 三次握手后的数据传输

2025-10-29 14:23:30.123456 IP 127.0.0.1.1883 > 127.0.0.1.54321: Flags [.], ack 58, win 65535, length 0
# ⬆️ 第一个 TCP Keep-Alive ACK 包（15 秒后）

2025-10-29 14:23:45.123456 IP 127.0.0.1.1883 > 127.0.0.1.54321: Flags [.], ack 58, win 65535, length 0
# ⬆️ 第二个 TCP Keep-Alive ACK 包（又过了 15 秒）

2025-10-29 14:24:00.123456 IP 127.0.0.1.1883 > 127.0.0.1.54321: Flags [.], ack 58, win 65535, length 0
# ⬆️ 第三个 TCP Keep-Alive ACK 包（再过了 15 秒）
```

**关键特征**：
- `Flags [.]` - 这是纯 ACK 包
- `length 0` - 没有数据载荷
- 间隔约 **15 秒**（这就是问题所在）

### MQTT Keep-Alive vs TCP Keep-Alive

观察时你会看到两种不同的包：

| 时间点 | 包类型 | 特征 | 间隔 |
|--------|--------|------|------|
| 15秒、30秒、45秒... | **TCP Keep-Alive** | `[.] ack`, `length 0` | **15 秒** |
| 60秒、120秒... | **MQTT PINGREQ/PINGRESP** | 有数据载荷的 MQTT 控制包 | **60 秒** |

## 停止测试

在测试程序终端按 `Ctrl+C` 即可停止。

## 配置说明

### 修改 MQTT Keep-Alive 周期

编辑 `main.go` 第 61 行：
```go
opts.SetKeepAlive(60 * time.Second)  // 改为你想要的时间
```

### 修改消息发送间隔

编辑 `main.go` 第 103 行：
```go
messageTicker := time.NewTicker(30 * time.Second)  // 改为你想要的间隔
```

## 问题说明

TCP Keep-Alive 15 秒是 **Go 语言 net 包的默认行为**，不是 GMQTT 的配置问题。

详细分析请查看：`docs/2025.10.29-TCP_KeepAlive问题分析.md`

## 依赖包

需要安装 paho.mqtt.golang：
```bash
go get github.com/eclipse/paho.mqtt.golang
```

或者使用项目已有的依赖：
```bash
cd /mnt/c/B-work/thingspanel-work/1-code/1-tp/thingspanel-gmqtt
go mod download
```
