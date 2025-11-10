package thingspanel

import (
	"context"
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


