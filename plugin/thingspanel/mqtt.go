package thingspanel

import (
	"fmt"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

type MqttClient struct {
	Client mqtt.Client
	IsFlag bool
	// 串行化通道：保证所有 Publish 按调用顺序执行，调用方不阻塞
	sendCh chan func()
}

var DefaultMqttClient *MqttClient = &MqttClient{}

func (c *MqttClient) MqttInit() error {
	opts := mqtt.NewClientOptions()
	opts.SetUsername("root")
	password := viper.GetString("mqtt.password")
	opts.SetPassword(password)
	addr := viper.GetString("mqtt.broker")
	if addr == "" {
		addr = "localhost:1883"
	}
	opts.AddBroker(addr)
	// 干净会话
	opts.SetCleanSession(true)
	// 失败重连
	opts.SetAutoReconnect(true)
	opts.SetConnectRetryInterval(1 * time.Second)   // 初始连接重试间隔
	opts.SetMaxReconnectInterval(200 * time.Second) // 丢失连接后的最大重试间隔

	opts.SetOrderMatters(true) // 保证消息有序，设备上下线状态必须有序

	opts.SetOnConnectHandler(func(c mqtt.Client) {
		fmt.Println("Mqtt客户端已连接")
	})
	opts.SetClientID("thingspanel-gmqtt-client")
	c.Client = mqtt.NewClient(opts)
	// 初始化串行化通道，启动后台发送协程
	c.sendCh = make(chan func(), 100)
	go c.sendWorker()
	for {
		if token := c.Client.Connect(); token.Wait() && token.Error() != nil {
			fmt.Println("Mqtt客户端连接失败(", addr, ")，等待重连...")
			time.Sleep(1 * time.Second)
		} else {
			fmt.Println("Mqtt客户端连接成功")
			c.IsFlag = true
			break
		}
	}
	return nil
}

func (c *MqttClient) SendData(topic string, data []byte) error {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("【SendData】异常捕捉：", err)
		}
	}()
	if !c.IsFlag {
		i := 1
		for {
			fmt.Println("等待...", i)
			if i == 10 || c.IsFlag {
				break
			}
			time.Sleep(1 * time.Second)
			i++
		}
	}
	if c.sendCh == nil {
		// 降级：通道未初始化时直接发布（不应该发生）
		token := c.Client.Publish(topic, 1, false, string(data))
		go func() {
			token.WaitTimeout(15 * time.Second)
		}()
		return nil
	}
	// 写入串行化通道，不阻塞调用方
	c.sendCh <- func() {
		token := c.Client.Publish(topic, 1, false, string(data))
		if !token.WaitTimeout(15 * time.Second) {
			Log.Warn("【消息发布超时】", zap.String("topic", topic), zap.String("data", string(data)))
			return
		}
		if err := token.Error(); err != nil {
			Log.Warn("【消息发布失败】", zap.String("topic", topic), zap.String("data", string(data)), zap.Error(err))
		}
	}
	return nil
}

// sendWorker 后台串行发送协程：按顺序执行所有发布任务
func (c *MqttClient) sendWorker() {
	for task := range c.sendCh {
		task()
	}
}
