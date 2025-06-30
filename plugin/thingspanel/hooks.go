package thingspanel

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/DrmagicE/gmqtt/plugin/thingspanel/util"
	"github.com/DrmagicE/gmqtt/server"
	"github.com/spf13/viper"
)

func (t *Thingspanel) HookWrapper() server.HookWrapper {
	return server.HookWrapper{
		OnBasicAuthWrapper:  t.OnBasicAuthWrapper,
		OnSubscribeWrapper:  t.OnSubscribeWrapper,
		OnMsgArrivedWrapper: t.OnMsgArrivedWrapper,
		OnConnectedWrapper:  t.OnConnectedWrapper,
		OnClosedWrapper:     t.OnClosedWrapper,
	}
}

func (t *Thingspanel) OnBasicAuthWrapper(pre server.OnBasicAuth) server.OnBasicAuth {
	return func(ctx context.Context, client server.Client, req *server.ConnectRequest) (err error) {
		//处理前一个插件的OnBasicAuth逻辑
		err = pre(ctx, client, req)
		if err != nil {
			Log.Error(err.Error())
			return err
		}
		if string(req.Connect.Username) == "root" {
			password := viper.GetString("mqtt.password")
			if string(req.Connect.Password) == password {
				return nil
			} else {
				err := errors.New("password error;")
				Log.Warn(err.Error())
				return err
			}
		}
		if string(req.Connect.Username) == "plugin" {
			password := viper.GetString("mqtt.plugin_password")
			if string(req.Connect.Password) == password {
				return nil
			} else {
				err := errors.New("password error;")
				Log.Warn(err.Error())
				return err
			}
		}
		// ... 处理本插件的鉴权逻辑
		Log.Info("鉴权Username：" + string(req.Connect.Username))
		Log.Info("鉴权Password：" + string(req.Connect.Password))

		// voucher是一个字符串，如果没有密码，voucher就是{"username":"xxx"}，如果有密码，voucher就是{"username":"xxx","password":"xxx"}
		voucher := ""
		if string(req.Connect.Password) != "" {
			voucher = fmt.Sprintf(`{"username":"%s","password":"%s"}`, string(req.Connect.Username), string(req.Connect.Password))
		} else {
			voucher = fmt.Sprintf(`{"username":"%s"}`, string(req.Connect.Username))
		}
		// 通过voucher验证设备
		Log.Debug("voucher: " + voucher)
		device, err := GetDeviceByVoucher(voucher)
		if err != nil {
			Log.Warn(err.Error())
			return err
		}
		Log.Info("设备Voucher：" + device.Voucher)
		Log.Info("ClientID：" + string(req.Connect.ClientID))
		// mqtt客户端id必须唯一
		err = SetStr("mqtt_clinet_id_"+string(req.Connect.ClientID), device.ID, 0)
		if err != nil {
			Log.Warn(err.Error())
			return err
		}
		return nil
	}
}

func (t *Thingspanel) OnConnectedWrapper(pre server.OnConnected) server.OnConnected {
	return func(ctx context.Context, client server.Client) {
		// 客户端连接后
		// 主题：device/status
		// 报文：{"token":username,"SYS_STATUS":"online"}
		// username为客户端用户名

		if client.ClientOptions().Username != "root" && client.ClientOptions().Username != "plugin" {
			deviceId, err := GetStr("mqtt_clinet_id_" + client.ClientOptions().ClientID)
			if err != nil {
				Log.Warn("获取设备ID失败")
				return
			}
			if deviceId == "" {
				Log.Warn("设备ID不存在")
				return
			}
			if err := DefaultMqttClient.SendData("devices/status/"+deviceId, []byte("1")); err != nil {
				Log.Warn("上报状态失败")
			}
			Log.Info("发送设备状态成功")
		}
	}
}
func (t *Thingspanel) OnClosedWrapper(pre server.OnClosed) server.OnClosed {
	return func(ctx context.Context, client server.Client, err error) {
		// 客户端断开连接后
		// 主题：device/status
		// 报文：{"token":username,"SYS_STATUS":"offline"}
		// username为客户端用户名
		if client.ClientOptions().Username != "root" || client.ClientOptions().Username != "plugin" {
			deviceId, err := GetStr("mqtt_clinet_id_" + client.ClientOptions().ClientID)
			if err != nil {
				Log.Warn("获取设备ID失败")
				return
			}
			if deviceId == "" {
				Log.Warn("设备ID不存在")
				return
			}
			if err := DefaultMqttClient.SendData("devices/status/"+deviceId, []byte("0")); err != nil {
				Log.Warn("上报状态失败")
			}
			Log.Info("发送设备状态成功")
		}
	}
}

// 订阅消息钩子函数
func (t *Thingspanel) OnSubscribeWrapper(pre server.OnSubscribe) server.OnSubscribe {
	return func(ctx context.Context, client server.Client, req *server.SubscribeRequest) error {
		username := client.ClientOptions().Username
		//root放行
		if username == "root" || username == "plugin" {
			return nil
		}

		the_sub := req.Subscribe.Topics[0].Name
		// 验证设备的订阅权限
		if !util.ValidateSubTopic(the_sub) {
			Log.Warn("订阅权限验证失败: " + the_sub)
			return errors.New("permission denied")
		}
		Log.Info("订阅权限验证成功: " + the_sub)
		return nil
	}
}

func (t *Thingspanel) OnMsgArrivedWrapper(pre server.OnMsgArrived) server.OnMsgArrived {
	return func(ctx context.Context, client server.Client, req *server.MsgArrivedRequest) (err error) {
		username := client.ClientOptions().Username
		Log.Info(fmt.Sprintf("OnMsgArrivedWrapper: username %s payload %s", username, string(req.Message.Payload)))
		// root用户和插件用户直接转发
		if username == "root" || username == "plugin" {
			RootMessageForwardWrapper(req.Message.Topic, req.Message.Payload, false)
			return nil
		}

		the_pub := string(req.Publish.TopicName)
		// 验证设备的发布权限
		if !util.ValidateTopic(the_pub) {
			return errors.New("permission denied")
		}

		// 后三位是/up的主题直接方放行【Mindjoy-MW】
		if the_pub[len(the_pub)-3:] == "/up" {
			return nil
		}

		// 这里按照需求重写custom/up/+
		// 检查是否为custom/up/+模式的主题
		if len(the_pub) > 10 && the_pub[:10] == "custom/up/" {
			// 提取子设备地址（主题中custom/up/后面的部分）
			subDeviceAddr := the_pub[10:]
			if subDeviceAddr != "" {
				// 解析原始payload为JSON
				var originalData interface{}
				if err := json.Unmarshal(req.Message.Payload, &originalData); err != nil {
					Log.Warn("解析custom/up消息payload失败: " + err.Error())
					return err
				}

				// 构造网关格式的payload
				gatewayPayload := map[string]interface{}{
					"sub_device_data": map[string]interface{}{
						subDeviceAddr: originalData,
					},
				}

				// 序列化为JSON并重写payload
				gatewayData, err := json.Marshal(gatewayPayload)
				if err != nil {
					Log.Warn("构造网关payload失败: " + err.Error())
					return err
				}

				// 重写消息内容
				req.Message.Payload = gatewayData
				// 重写主题为gateway/telemetry
				the_pub = "gateway/telemetry"
				// 不修改req.Publish.TopicName，保持原始主题用于后续判断

				Log.Info(fmt.Sprintf("成功重写custom/up消息为网关格式: %s", subDeviceAddr))
			}
		}

		// 消息重写
		newMsgMap := make(map[string]interface{})
		deviceId, err := GetStr("mqtt_clinet_id_" + client.ClientOptions().ClientID)
		if err != nil {
			return err
		}
		newMsgMap["device_id"] = deviceId
		newMsgMap["values"] = req.Message.Payload
		newMsgJson, _ := json.Marshal(newMsgMap)
		req.Message.Payload = newMsgJson
		// 如果原主题被转换，丢弃消息，重新发布到转换后的主题
		if the_pub != string(req.Publish.TopicName) {
			DefaultMqttClient.SendData(the_pub, req.Message.Payload)
			return errors.New("message is discarded;")
		}
		return nil
	}
}
