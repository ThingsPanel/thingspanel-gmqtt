package thingspanel

import (
	"context"
	"strings"

	"go.uber.org/zap"
)

type TopicMapService struct{}

func NewTopicMapService() *TopicMapService {
	return &TopicMapService{}
}

// ResolveUpTarget tries to resolve an up-direction target topic for a given device_config_id and incoming source topic.
// Returns target topic and true if matched; otherwise returns empty string and false.
func (s *TopicMapService) ResolveUpTarget(ctx context.Context, deviceConfigID string, incomingSource string) (string, bool) {
	mappings, err := GetMappingsWithCache(ctx, deviceConfigID, DirectionUp)
	if err != nil || len(mappings) == 0 {
		return "", false
	}
	for _, m := range mappings {
		rx, ok := compileSourcePattern(m.SourceTopic)
		if !ok {
			continue
		}
		if rx.MatchString(incomingSource) {
			return applyTarget(m.TargetTopic, incomingSource), true
		}
	}
	return "", false
}

// AllowDownSubscribe returns true if a subscribe topic is allowed by down-direction custom mappings.
// 按设计，设备订阅的是“下行原始主题”，因此应当匹配 source_topic。
func (s *TopicMapService) AllowDownSubscribe(ctx context.Context, deviceConfigID string, subscribeTopic string) bool {
	mappings, err := GetMappingsWithCache(ctx, deviceConfigID, DirectionDown)
	if err != nil || len(mappings) == 0 {
		return false
	}
	for _, m := range mappings {
		rx, ok := compileSourcePattern(m.SourceTopic)
		if !ok {
			continue
		}
		if rx.MatchString(subscribeTopic) {
			return true
		}
	}
	return false
}

// ResolveDownSource returns a concrete original device topic (source_topic rendered) 解析下行原始主题
// when platform publishes to a normalized down target topic. 平台发布到规范化下行目标主题时，解析下行原始主题
// variables currently support: device_number 目前支持的变量：device_number
func (s *TopicMapService) ResolveDownSource(ctx context.Context, deviceConfigID string, normalizedTarget string, deviceNumber string) (string, bool) {
	// 获取设备配置ID对应的下行自定义主题映射
	mappings, err := GetMappingsWithCache(ctx, deviceConfigID, DirectionDown)
	if err != nil || len(mappings) == 0 {
		return "", false
	}
	// 遍历下行自定义主题映射，逐条尝试匹配规范化下行目标主题
	for _, m := range mappings {
		// 编译目标主题模式，例如：devices/telemetry/control/{device_number} 编译成正则表达式
		// 编译成正则表达式后，可以匹配规范化下行目标主题，例如：devices/telemetry/control/123456
		rx, ok := compileTargetPattern(m.TargetTopic)
		// 如果编译失败，则跳过
		if !ok {
			Log.Debug("【下行自定义主题额外转发】编译目标主题模式失败", zap.String("target_topic", m.TargetTopic))
			continue
		}
		matched := rx.MatchString(normalizedTarget)
		if matched {
			vars := map[string]string{
				"device_number": deviceNumber,
			}
			src := renderTopicFromTemplate(m.SourceTopic, vars)
			Log.Debug("【下行自定义主题额外转发】渲染后的原始主题", zap.String("rendered_source", src))
			// If '+' remains, we cannot derive a concrete target. Skip such rules. 如果存在+，则无法推导出具体的主题，跳过此类规则。
			if strings.Contains(src, "+") || strings.Contains(src, "#") {
				Log.Debug("【下行自定义主题额外转发】渲染后的主题包含通配符，跳过", zap.String("rendered_source", src))
				continue
			}
			return src, true
		}
	}
	return "", false
}
