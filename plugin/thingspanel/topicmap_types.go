package thingspanel

import "time"

// Direction represents mapping direction: "up" or "down".
type Direction string

const (
	DirectionUp   Direction = "up"
	DirectionDown Direction = "down"
)

// DeviceTopicMapping maps a device's source topic to a platform target topic.
// Mirrors the PostgreSQL table `device_topic_mappings`.
type DeviceTopicMapping struct {
	ID             int64      `gorm:"column:id;primaryKey"`
	DeviceConfigID string     `gorm:"column:device_config_id"`
	Name           string     `gorm:"column:name"`
	Direction      string     `gorm:"column:direction"`
	SourceTopic    string     `gorm:"column:source_topic"`
	TargetTopic    string     `gorm:"column:target_topic"`
	DataIdentifier *string    `gorm:"column:data_identifier"`
	Priority       int        `gorm:"column:priority"`
	Enabled        bool       `gorm:"column:enabled"`
	Description    *string    `gorm:"column:description"`
	CreatedAt      *time.Time `gorm:"column:created_at"`
	UpdatedAt      *time.Time `gorm:"column:updated_at"`
}

func (DeviceTopicMapping) TableName() string {
	return "device_topic_mappings"
}
