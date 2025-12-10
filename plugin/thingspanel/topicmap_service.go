package thingspanel

import (
	"context"
	"encoding/json"
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
// payload will be trimmed to params when data_identifier matched; otherwise kept as-is.
func (s *TopicMapService) ResolveDownSource(ctx context.Context, deviceConfigID string, normalizedTarget string, deviceNumber string, payload []byte) (string, []byte, bool) {
	// 获取设备配置ID对应的下行自定义主题映射
	mappings, err := GetMappingsWithCache(ctx, deviceConfigID, DirectionDown)
	if err != nil || len(mappings) == 0 {
		return "", nil, false
	}

	// 兜底配置（data_identifier 为空）
	fallbackSource := ""
	fallbackPayload := payload

	// 遍历下行自定义主题映射，逐条尝试匹配规范化下行目标主题
	for _, m := range mappings {
		rx, ok := compileTargetPattern(m.TargetTopic)
		if !ok {
			Log.Debug("【下行自定义主题额外转发】编译目标主题模式失败", zap.String("target_topic", m.TargetTopic))
			continue
		}
		if !rx.MatchString(normalizedTarget) {
			continue
		}

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

		// 标识符匹配
		if m.DataIdentifier != nil && strings.TrimSpace(*m.DataIdentifier) != "" {
			var cmd struct {
				Method string          `json:"method"`
				Params json.RawMessage `json:"params"`
			}
			if err := json.Unmarshal(payload, &cmd); err != nil {
				Log.Warn("【下行自定义主题额外转发】payload 解析失败，跳过标识符匹配", zap.Error(err))
				continue
			}
			if cmd.Method != strings.TrimSpace(*m.DataIdentifier) {
				continue
			}
			// data_identifier 匹配成功，payload 仅保留 params
			out := cmd.Params
			if len(out) == 0 {
				out = []byte("{}")
			}
			return src, out, true
		}

		// 兜底：记录第一个 data_identifier 为空的匹配
		if fallbackSource == "" {
			fallbackSource = src
		}
	}

	if fallbackSource != "" {
		return fallbackSource, fallbackPayload, true
	}
	return "", nil, false
}
