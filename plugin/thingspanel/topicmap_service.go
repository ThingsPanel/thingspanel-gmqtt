package thingspanel

import (
	"context"
	"strings"
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

// ResolveDownSource returns a concrete original device topic (source_topic rendered)
// when platform publishes to a normalized down target topic.
// variables currently support: device_number
func (s *TopicMapService) ResolveDownSource(ctx context.Context, deviceConfigID string, normalizedTarget string, deviceNumber string) (string, bool) {
	mappings, err := GetMappingsWithCache(ctx, deviceConfigID, DirectionDown)
	if err != nil || len(mappings) == 0 {
		return "", false
	}
	for _, m := range mappings {
		rx, ok := compileTargetPattern(m.TargetTopic)
		if !ok {
			continue
		}
		if rx.MatchString(normalizedTarget) {
			vars := map[string]string{
				"device_number": deviceNumber,
			}
			src := renderTopicFromTemplate(m.SourceTopic, vars)
			// If '+' remains, we cannot derive a concrete target. Skip such rules.
			if strings.Contains(src, "+") || strings.Contains(src, "#") {
				continue
			}
			return src, true
		}
	}
	return "", false
}
