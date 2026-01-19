package thingspanel

import (
	"encoding/json"
	"errors"
	"time"

	"gopkg.in/redis.v5"
)

const (
	devDebugCfgKeyPrefix  = "tp:devdebug:cfg:"
	devDebugLogsKeyPrefix = "tp:devdebug:logs:"
)

var devDebugNow = time.Now

type DeviceDebugConfig struct {
	Enabled         bool  `json:"enabled"`
	ExpireAt        int64 `json:"expire_at"`
	MaxItems        int   `json:"max_items"`
	PayloadMaxBytes int   `json:"payload_max_bytes"`
}

type DeviceDebugLogEntry struct {
	Ts        string                 `json:"ts"`
	Event     string                 `json:"event"`     // auth|subscribe|publish|forward
	Direction string                 `json:"direction"` // up|down|na
	DeviceID  string                 `json:"device_id"`
	ClientID  string                 `json:"client_id,omitempty"`
	Username  string                 `json:"username,omitempty"`
	Topic     string                 `json:"topic,omitempty"`
	Payload   string                 `json:"payload,omitempty"`
	Result    string                 `json:"result"` // ok|denied|error|discarded
	Error     string                 `json:"error,omitempty"`
	Extra     map[string]interface{} `json:"extra,omitempty"`
}

func devDebugCfgKey(deviceID string) string {
	return devDebugCfgKeyPrefix + deviceID
}

func devDebugLogsKey(deviceID string) string {
	return devDebugLogsKeyPrefix + deviceID
}

func GetDeviceDebugConfig(deviceID string) (DeviceDebugConfig, bool, error) {
	if deviceID == "" {
		return DeviceDebugConfig{}, false, errors.New("empty device_id")
	}
	if redisCache == nil {
		return DeviceDebugConfig{}, false, errors.New("redis not initialized")
	}
	var cfg DeviceDebugConfig
	if err := GetRedisForJsondata(devDebugCfgKey(deviceID), &cfg); err != nil {
		if err == redis.Nil {
			return DeviceDebugConfig{}, false, nil
		}
		return DeviceDebugConfig{}, false, err
	}
	if !cfg.Enabled {
		return cfg, false, nil
	}
	if cfg.ExpireAt > 0 && devDebugNow().Unix() > cfg.ExpireAt {
		return cfg, false, nil
	}
	if cfg.MaxItems <= 0 {
		cfg.MaxItems = 1000
	}
	if cfg.PayloadMaxBytes < 0 {
		cfg.PayloadMaxBytes = 0
	}
	return cfg, true, nil
}

// WriteDeviceDebugLog appends a log entry if device debug is enabled.
// It is safe to call frequently; missing/expired config results in a no-op.
func WriteDeviceDebugLog(deviceID string, entry DeviceDebugLogEntry) (bool, error) {
	cfg, enabled, err := GetDeviceDebugConfig(deviceID)
	if err != nil || !enabled {
		return false, err
	}
	if redisCache == nil {
		return false, errors.New("redis not initialized")
	}
	entry.DeviceID = deviceID
	if entry.Ts == "" {
		entry.Ts = devDebugNow().Format(time.RFC3339Nano)
	}

	if cfg.PayloadMaxBytes <= 0 {
		entry.Payload = ""
	} else if len(entry.Payload) > cfg.PayloadMaxBytes {
		entry.Payload = entry.Payload[:cfg.PayloadMaxBytes]
		if entry.Extra == nil {
			entry.Extra = map[string]interface{}{}
		}
		entry.Extra["payload_truncated"] = true
	}

	raw, err := json.Marshal(entry)
	if err != nil {
		return false, err
	}

	logsKey := devDebugLogsKey(deviceID)
	pipe := redisCache.Pipeline()
	pipe.LPush(logsKey, raw)
	pipe.LTrim(logsKey, 0, int64(cfg.MaxItems-1))
	if cfg.ExpireAt > 0 {
		ttlSeconds := (cfg.ExpireAt - devDebugNow().Unix()) + 10*60
		if ttlSeconds > 0 {
			pipe.Expire(logsKey, time.Duration(ttlSeconds)*time.Second)
		}
	}
	_, err = pipe.Exec()
	if err == redis.Nil {
		return false, nil
	}
	return err == nil, err
}
