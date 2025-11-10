package thingspanel

import (
	"context"
	"fmt"
	"time"
)

// Cache key helpers (keep consistent with docs)
func cacheKeyUp(deviceConfigID string) string {
	return fmt.Sprintf("tp:topicmap:up:%s", deviceConfigID)
}
func cacheKeyDown(deviceConfigID string) string {
	return fmt.Sprintf("tp:topicmap:down:%s", deviceConfigID)
}

// GetMappingsWithCache gets mappings using Redis cache; if miss, loads from PG and sets cache.
func GetMappingsWithCache(ctx context.Context, deviceConfigID string, direction Direction) ([]DeviceTopicMapping, error) {
	var key string
	if direction == DirectionUp {
		key = cacheKeyUp(deviceConfigID)
	} else {
		key = cacheKeyDown(deviceConfigID)
	}
	var cached []DeviceTopicMapping
	if err := GetRedisForJsondata(key, &cached); err == nil && len(cached) > 0 {
		return cached, nil
	}
	rows, err := LoadEnabledMappings(ctx, deviceConfigID, direction)
	if err != nil {
		return nil, err
	}
	// cache without TTL (per docs) or a long TTL as a fallback
	_ = SetRedisForJsondata(key, rows, 24*time.Hour)
	return rows, nil
}

// InvalidateMappingCache clears cache for a device_config_id on mapping CRUD.
func InvalidateMappingCache(deviceConfigID string) {
	_ = DelKey(cacheKeyUp(deviceConfigID))
	_ = DelKey(cacheKeyDown(deviceConfigID))
}
