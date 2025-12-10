package thingspanel

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/DrmagicE/gmqtt/plugin/thingspanel/util"
	"github.com/DrmagicE/gmqtt/server"
	"github.com/spf13/viper"
	"go.uber.org/zap"
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
		Log.Info("【鉴权】开始",
			zap.String("username", string(req.Connect.Username)),
			zap.String("password", string(req.Connect.Password)),
			zap.String("client_id", string(req.Connect.ClientID)))

		// voucher是一个字符串，如果没有密码，voucher就是{"username":"xxx"}，如果有密码，voucher就是{"username":"xxx","password":"xxx"}
		voucher := ""
		if string(req.Connect.Password) != "" {
			voucher = fmt.Sprintf(`{"username":"%s","password":"%s"}`, string(req.Connect.Username), string(req.Connect.Password))
		} else {
			voucher = fmt.Sprintf(`{"username":"%s"}`, string(req.Connect.Username))
		}
		// 通过voucher验证设备
		device, err := GetDeviceByVoucher(voucher)
		if err != nil {
			Log.Warn("【鉴权】失败",
				zap.String("client_id", string(req.Connect.ClientID)),
				zap.Error(err))
			return err
		} else {
			Log.Info("【鉴权】通过",
				zap.String("client_id", string(req.Connect.ClientID)),
				zap.String("device_id", device.ID))
		}
		// mqtt客户端id必须唯一
		err = SetStr("mqtt_clinet_id_"+string(req.Connect.ClientID), device.ID, 0)
		if err != nil {
			Log.Error(err.Error())
			return err
		}
		return nil
	}
}

// 设备上线钩子函数
func (t *Thingspanel) OnConnectedWrapper(pre server.OnConnected) server.OnConnected {
	return func(ctx context.Context, client server.Client) {
		// 客户端连接后
		// 主题：device/status
		// 报文：{"token":username,"SYS_STATUS":"online"}
		// username为客户端用户名

		if client.ClientOptions().Username != "root" && client.ClientOptions().Username != "plugin" {
			deviceId, err := GetStr("mqtt_clinet_id_" + client.ClientOptions().ClientID)
			if err != nil {
				Log.Warn("【上线回调】获取设备ID失败", zap.String("client_id", client.ClientOptions().ClientID), zap.Error(err))
				return
			}
			if deviceId == "" {
				Log.Warn("【上线回调】设备ID不存在", zap.String("client_id", client.ClientOptions().ClientID))
				return
			}
			if err := DefaultMqttClient.SendData("devices/status/"+deviceId, []byte("1")); err != nil {
				Log.Warn("【设备上线】上报状态失败", zap.String("device_id", deviceId), zap.Error(err))
			}
		}
	}
}
func (t *Thingspanel) OnClosedWrapper(pre server.OnClosed) server.OnClosed {
	return func(ctx context.Context, client server.Client, err error) {
		// 客户端断开连接后
		// 主题：device/status
		// 报文：{"token":username,"SYS_STATUS":"offline"}
		// username为客户端用户名
		Log.Info("【连接断开】OnClosedWrapper",
			zap.String("username", client.ClientOptions().Username),
			zap.String("client_id", client.ClientOptions().ClientID),
			zap.Error(err))
		if client.ClientOptions().Username != "root" && client.ClientOptions().Username != "plugin" {
			deviceId, err := GetStr("mqtt_clinet_id_" + client.ClientOptions().ClientID)
			if err != nil {
				Log.Warn("【连接断开】获取设备ID失败",
					zap.String("client_id", client.ClientOptions().ClientID),
					zap.Error(err))
				return
			}
			if deviceId == "" {
				Log.Warn("【连接断开】设备ID不存在",
					zap.String("client_id", client.ClientOptions().ClientID))
				return
			}
			if err := DefaultMqttClient.SendData("devices/status/"+deviceId, []byte("0")); err != nil {
				Log.Warn("【连接断开】上报状态失败",
					zap.String("client_id", client.ClientOptions().ClientID),
					zap.Error(err))
			}
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
		// 验证设备的订阅权限；若失败，尝试下行自定义映射放行
		if !util.ValidateSubTopic(the_sub) {
			// 获取设备与配置ID
			deviceId, err := GetStr("mqtt_clinet_id_" + client.ClientOptions().ClientID)
			if err == nil && deviceId != "" {
				if dev, derr := GetDeviceById(deviceId); derr == nil && dev != nil && dev.DeviceConfigID != nil {
					svc := NewTopicMapService()
					if svc.AllowDownSubscribe(ctx, *dev.DeviceConfigID, the_sub) {
						Log.Info("【自定义订阅】通过（自定义下行映射）", zap.String("topic", the_sub))
						return nil
					}
				}
			}
			Log.Warn("【订阅】权限验证失败", zap.String("topic", the_sub), zap.String("client_id", client.ClientOptions().ClientID), zap.Error(err))
			return errors.New("permission denied")
		}
		return nil
	}
}

func (t *Thingspanel) OnMsgArrivedWrapper(pre server.OnMsgArrived) server.OnMsgArrived {
	return func(ctx context.Context, client server.Client, req *server.MsgArrivedRequest) (err error) {
		username := client.ClientOptions().Username
		Log.Debug("【收到消息】OnMsgArrivedWrapper",
			zap.String("topic", req.Message.Topic),
			zap.String("client_id", client.ClientOptions().ClientID),
			zap.String("username", username),
			zap.String("payload", string(req.Message.Payload)))
		// root用户和插件用户直接转发
		if username == "root" || username == "plugin" {
			// RootMessageForwardWrapper(req.Message.Topic, req.Message.Payload, false)
			// root平台下发：若主题属于规范“下行主题”，提取设备号并按映射额外转发到设备原始主题
			topic := req.Message.Topic
			if deviceNumber, ok := TryExtractDeviceNumberFromNormalized(topic); ok && deviceNumber != "" {
				if dev, derr := GetDeviceByNumber(deviceNumber); derr == nil && dev != nil && dev.DeviceConfigID != nil {
					svc := NewTopicMapService()
					if src, outPayload, matched := svc.ResolveDownSource(ctx, *dev.DeviceConfigID, topic, deviceNumber, req.Message.Payload); matched && src != "" {
						if err := DefaultMqttClient.SendData(src, outPayload); err != nil {
							Log.Warn("【下行自定义主题额外转发】失败", zap.String("topic", topic), zap.String("client_id", client.ClientOptions().ClientID), zap.Error(err))
						} else {
							Log.Info("【下行自定义主题额外转发】成功", zap.String("topic", topic), zap.String("client_id", client.ClientOptions().ClientID), zap.String("target", src))
						}
					}
				}
			}
			return nil
		}

		the_pub := string(req.Publish.TopicName)

		// 获取设备与配置ID（用于自定义映射）
		deviceId, err := GetStr("mqtt_clinet_id_" + client.ClientOptions().ClientID)
		if err != nil {
			return err
		}
		var deviceConfigID string
		if deviceId != "" {
			if dev, derr := GetDeviceById(deviceId); derr == nil && dev != nil && dev.DeviceConfigID != nil {
				deviceConfigID = *dev.DeviceConfigID
			}
		}

		// 优先尝试上行自定义映射
		if deviceConfigID != "" {
			svc := NewTopicMapService()
			if target, ok := svc.ResolveUpTarget(ctx, deviceConfigID, the_pub); ok && target != "" {
				newMsgMap := make(map[string]interface{})
				newMsgMap["device_id"] = deviceId
				newMsgMap["values"] = req.Message.Payload
				newMsgJson, _ := json.Marshal(newMsgMap)
				if err := DefaultMqttClient.SendData(target, newMsgJson); err != nil {
					Log.Warn("【上行自定义主题转发】失败", zap.String("topic", the_pub), zap.String("client_id", client.ClientOptions().ClientID), zap.Error(err))
					return nil
				}
				Log.Info("【上行自定义主题转发】成功", zap.String("topic", the_pub), zap.String("client_id", client.ClientOptions().ClientID), zap.String("target", target))
				// 丢弃原消息
				return errors.New("message is discarded;")
			} else {
				Log.Debug("【上行】未匹配到自定义主题", zap.String("topic", the_pub), zap.String("client_id", client.ClientOptions().ClientID))
			}
		}

		// 验证设备的发布权限；若失败直接拒绝
		if !util.ValidateTopic(the_pub) {
			Log.Warn("【上行】权限验证失败", zap.String("topic", the_pub), zap.String("client_id", client.ClientOptions().ClientID))
			return errors.New("permission denied")
		}

		// 后三位是/up的主题直接方放行【Mindjoy-MW】
		// if the_pub[len(the_pub)-3:] == "/up" {
		// 	return nil
		// }

		// 消息重写
		newMsgMap := make(map[string]interface{})
		newMsgMap["device_id"] = deviceId
		newMsgMap["values"] = req.Message.Payload
		newMsgJson, _ := json.Marshal(newMsgMap)
		req.Message.Payload = newMsgJson
		return nil
	}
}
