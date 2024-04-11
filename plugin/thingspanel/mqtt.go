package thingspanel

import (
	"fmt"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/spf13/viper"
)

type MqttClient struct {
	Client mqtt.Client
	IsFlag bool
}

var DefaultMqttClient *MqttClient = &MqttClient{}

func (c *MqttClient) MqttInit() error {
	opts := mqtt.NewClientOptions()
	opts.SetUsername("root")
	password := viper.GetString("mqtt.password")
	opts.SetPassword(password)
	opts.AddBroker("localhost:1883")
	// 干净会话
	opts.SetCleanSession(true)
	// 失败重连
	opts.SetAutoReconnect(true)
	opts.SetConnectRetryInterval(1 * time.Second)   // 初始连接重试间隔
	opts.SetMaxReconnectInterval(200 * time.Second) // 丢失连接后的最大重试间隔

	opts.SetOrderMatters(false) //设置消息的顺序
	//opts.OnConnectionLost = connectLostHandler
	opts.SetOnConnectHandler(func(c mqtt.Client) {
		fmt.Println("Mqtt客户端已连接")
	})
	opts.SetClientID("thingspanel-gmqtt-client")
	c.Client = mqtt.NewClient(opts)
	// 等待连接成功
	// 等待连接成功
	for {
		if token := c.Client.Connect(); token.Wait() && token.Error() != nil {
			fmt.Println("Mqtt客户端连接失败，等待重连...")
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
			return
		}
	}()
	//go func() {
	Log.Info("检查MqttClIent连接状态...")
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
	Log.Info("发送设备状态...")
	token := c.Client.Publish(topic, 0, false, string(data))
	if !token.WaitTimeout(5 * time.Second) {
		Log.Warn("发送设备状态超时")
	} else if err := token.Error(); err != nil {
		Log.Warn("发送设备状态失败: " + err.Error())
	}
	Log.Info("发送设备状态完成")
	//}()
	return nil
}
