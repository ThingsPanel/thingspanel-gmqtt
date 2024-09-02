package util

import (
	"strings"
)

var subList = []string{
	"devices/telemetry/control/{device_number}",   //订阅平台下发的控制
	"devices/telemetry/control/{device_number}/+", //订阅平台下发的控制
	"devices/attributes/set/{device_number}/+",    //订阅平台下发的属性设置
	"devices/attributes/get/{device_number}",      //订阅平台对属性的请求
	"devices/command/{device_number}/+",           //订阅命令

	"ota/devices/infrom/{device_number}", //接收升级任务（固件升级相关）

	"devices/attributes/response/{device_number}/+", //订阅平台收到属性的响应
	"devices/event/response/{device_number}/+",      //接收平台收到事件的响应

	"gateway/telemetry/control/{device_number}", //订阅平台下发的控制(网关)
	"gateway/attributes/set/{device_number}/+",  //订阅平台下发的属性设置(网关)
	"gateway/attributes/get/{device_number}",    //订阅平台对属性的请求(网关)
	"gateway/command/{device_number}/+",         //订阅命令(网关)

	"gateway/attributes/response/{device_number}/+", //订阅平台收到属性的响应(网关)
	"gateway/event/response/{device_number}/+",      //接收平台收到事件的响应(网关)

	"{device_number}/down", //心智悦喷淋一体机下行数据

	"devices/register/response/+",    //网关子设备注册平台回复
	"devices/config/down/response/+", //设备配置下载平台回复
}

// ValidateTopic 检查一个主题是否符合subList里的任何一种模式
func ValidateSubTopic(topic string) bool {
	for _, pattern := range subList {
		if matchesPatternSub(topic, pattern) {
			return true
		}
	}
	return false
}

// matchesPattern 检查一个主题是否符合给定的模式
func matchesPatternSub(topic, pattern string) bool {
	topicParts := strings.Split(topic, "/")
	patternParts := strings.Split(pattern, "/")

	// 如果主题和模式部分的长度不一致，则不匹配
	if len(topicParts) != len(patternParts) {
		return false
	}

	// 检查每个部分
	for i := range topicParts {
		switch patternParts[i] {
		case "{device_number}":
			// {device_number} 部分不能是+或者#
			if topicParts[i] == "+" || topicParts[i] == "#" {
				return false
			}
		case "+":
			// +部分不可以是#通配符，可以是其他任意字符包括+通配符
			if topicParts[i] == "#" {
				return false
			}
		default:
			// 其他部分必须相等
			if topicParts[i] != patternParts[i] {
				return false
			}
		}
	}

	return true
}
