package thingspanel

import (
	"context"
	"errors"
	"sort"
)

// LoadEnabledMappings loads enabled mappings for a device_config_id and direction, sorted by priority ASC.
func LoadEnabledMappings(ctx context.Context, deviceConfigID string, direction Direction) ([]DeviceTopicMapping, error) {
	if deviceConfigID == "" {
		return nil, errors.New("empty deviceConfigID")
	}
	var rows []DeviceTopicMapping
	tx := db.WithContext(ctx).
		Model(&DeviceTopicMapping{}).
		Where("device_config_id = ? AND direction = ? AND enabled = true", deviceConfigID, string(direction)).
		Find(&rows)
	if tx.Error != nil {
		return nil, tx.Error
	}
	// Ensure ascending priority
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].Priority < rows[j].Priority })
	return rows, nil
}
