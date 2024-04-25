package util

import (
	"strings"
)

// 物联网设备主题规则列表
var pubList = []string{
	"devices/telemetry/control",         // 遥测上报
	"devices/attributes/+",              // 属性上报
	"devices/event/+",                   // 事件上报
	"ota/device/progress",               // 设备升级进度更新
	"devices/attributes/set/response/+", // 属性设置响应上报
	"devices/command/response/+",        // 命令响应上报

	"gateway/telemetry/control",         // 设备遥测（网关）
	"gateway/attributes/+",              // 属性上报 （网关）
	"gateway/event/+",                   // 事件上报 （网关）
	"gateway/attributes/set/response/+", // 属性设置响应上报 （网关）
	"gateway/command/response/+",        // 命令响应上报 （网关）

	"+/up", //心智悦喷淋一体机上行数据
}

// MQTT 通配符
const mqttWildcard = "+"

// ValidateTopic 检查一个主题是否符合pubList里的任何一种模式
func ValidateTopic(topic string) bool {
	for _, pattern := range pubList {
		if matchesPattern(topic, pattern) {
			return true
		}
	}
	return false
}

// matchesPattern 检查一个主题是否符合给定的模式
func matchesPattern(topic, pattern string) bool {
	topicParts := strings.Split(topic, "/")
	patternParts := strings.Split(pattern, "/")

	// 如果主题和模式部分的长度不一致，则不匹配
	if len(topicParts) != len(patternParts) {
		return false
	}

	// 检查是否直接匹配或者是通配符
	for i := range topicParts {
		if patternParts[i] != mqttWildcard && topicParts[i] != patternParts[i] {
			return false
		}
	}

	return true
}
