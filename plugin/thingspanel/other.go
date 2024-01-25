package thingspanel

import (
	"errors"
	"fmt"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
)

// 允许订阅的主题列表
var AllowSubscribeTopicList = [2]string{
	"+/devices/${username}/sys/commands/+",       // 命令下发(华为云物联网平台规范)
	"+/devices/${username}/sys/properties/set/+", // 属性设置(华为云物联网平台规范)
}

// 主题转换规则
var TopicConvertMap = map[string]string{
	"+/devices/${username}/sys/properties/report": "device/attributes", // 属性上报(华为云物联网平台规范)
	"+/devices/${username}/sys/events/up":         "device/event",      // 事件上报(华为云物联网平台规范)
}

// 订阅主题时的回调函数，用于判断是否允许订阅
func OtherOnSubscribeWrapper(topic string, username string) error {
	fmt.Println("OtherOnSubscribeWrapper--" + topic)
	//允许上报属性的主题：+/devices/+/sys/properties/report
	//允许上报事件的主题：+/devices/+/sys/events
	// 如果topic属于topicList种的主题，则允许订阅，注意：${}中的是变量，+是mqtt通配符
	for _, v := range AllowSubscribeTopicList {
		// 将${username}替换为username
		v = strings.Replace(v, "${username}", username, -1)
		// 将+替换为[^/]+，将#替换为.*，然后使用正则表达式匹配
		reg := strings.Replace(v, "+", "[^/]+", -1)
		reg = strings.Replace(reg, "#", ".*", -1)
		match, _ := regexp.MatchString(reg, topic)
		if match {
			return nil
		} else {
			fmt.Println("topic not allowed--")
			return errors.New("topic not allowed")
		}
	}
	return errors.New("topic not allowed")

}

// 收到消息时的回调函数，用于判断是否允许收到消息
// 可在此处做主题转换
func OtherOnMsgArrivedWrapper(topic string, username string) (string, error) {
	// 如果topic属于topicConvertMap种的主题，则转换为对应的主题
	for k, v := range TopicConvertMap {
		// 将${username}替换为username
		k = strings.Replace(k, "${username}", username, -1)
		// 将+替换为[^/]+，将#替换为.*，然后使用正则表达式匹配
		reg := strings.Replace(k, "+", "[^/]+", -1)
		reg = strings.Replace(reg, "#", ".*", -1)
		match, _ := regexp.MatchString(reg, topic)
		if match {
			return v, nil
		} else {
			return topic, errors.New("topic not allowed")
		}
	}
	return topic, errors.New("topic not allowed")
}

// 如果root用户的消息的主题是device/attributes/+,择转发消息
// flag为true时，转发消息
func RootMessageForwardWrapper(topic string, payload []byte, flag bool) error {
	// 如果flag为false，则直接返回
	if !flag {
		return nil
	}
	var username string
	var topicConvertMap = map[string]string{
		"device/attributes/+": "mindjob/devices/${username}/sys/properties/set/request_id=", // 命令下发(华为云物联网平台规范)
	}
	// 如果topic属于topicConvertMap的key，则转换为对应的value
	for k, v := range topicConvertMap {
		// 将+替换为[^/]+，将#替换为.*，然后使用正则表达式匹配
		reg := strings.Replace(k, "+", "[^/]+", -1)
		reg = strings.Replace(reg, "#", ".*", -1)
		match, _ := regexp.MatchString(reg, topic)
		if match {
			// 获取username
			username = strings.Split(topic, "/")[2]
			// 将${username}替换为username
			v = strings.Replace(v, "${username}", username, -1)
			// 获取随机的6位字符串数字，不使用RandomString
			request_id := strconv.Itoa(rand.Intn(899999) + 100000)
			// 给v添加request_id,request_id的值为随机6位数字
			v = v + request_id
			// 转发消息
			fmt.Println("RootMessageForwardWrapper--" + v)
			if err := DefaultMqttClient.SendData(v, payload); err != nil {
				return err
			}
		}
	}
	return nil
}
