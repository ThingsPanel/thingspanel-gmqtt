package thingspanel

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"gopkg.in/redis.v5"
)

func TestWriteDeviceDebugLog_NoConfigNoop(t *testing.T) {
	s, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	defer s.Close()

	redisCache = redis.NewClient(&redis.Options{Addr: s.Addr()})
	t.Cleanup(func() { _ = redisCache.Close() })

	devDebugNow = func() time.Time { return time.Unix(1000, 0) }
	t.Cleanup(func() { devDebugNow = time.Now })

	wrote, err := WriteDeviceDebugLog("dev1", DeviceDebugLogEntry{Action: "publish", Direction: "up", Outcome: "ok"})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if wrote {
		t.Fatalf("expected wrote=false")
	}
	got, err := s.List(devDebugLogsKey("dev1"))
	if err != nil {
		// miniredis returns an error for missing keys
		return
	}
	if len(got) != 0 {
		t.Fatalf("expected no logs, got %d", len(got))
	}
}

func TestWriteDeviceDebugLog_TruncateAndTrimAndTTL(t *testing.T) {
	s, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	defer s.Close()

	redisCache = redis.NewClient(&redis.Options{Addr: s.Addr()})
	t.Cleanup(func() { _ = redisCache.Close() })

	devDebugNow = func() time.Time { return time.Unix(2000, 0) }
	t.Cleanup(func() { devDebugNow = time.Now })

	deviceID := "dev1"
	cfg := DeviceDebugConfig{
		Enabled:         true,
		ExpireAt:        devDebugNow().Unix() + 60,
		MaxItems:        2,
		PayloadMaxBytes: 5,
	}
	if err := SetRedisForJsondata(devDebugCfgKey(deviceID), cfg, 0); err != nil {
		t.Fatalf("set cfg: %v", err)
	}

	for i := 0; i < 3; i++ {
		_, err := WriteDeviceDebugLog(deviceID, DeviceDebugLogEntry{
			Action:    "publish",
			Direction: "up",
			Payload:   "abcdefghij",
			Outcome:   "ok",
			Meta: map[string]interface{}{
				"client_id": "c1",
				"username":  "u1",
				"topic":     "t",
			},
		})
		if err != nil {
			t.Fatalf("write log: %v", err)
		}
	}

	logKey := devDebugLogsKey(deviceID)
	got, err := s.List(logKey)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected trimmed to 2, got %d", len(got))
	}

	var entry DeviceDebugLogEntry
	if err := json.Unmarshal([]byte(got[0]), &entry); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if entry.Payload != "abcde" {
		t.Fatalf("expected payload truncated, got %q", entry.Payload)
	}
	if entry.Meta == nil || entry.Meta["payload_truncated"] != true {
		t.Fatalf("expected payload_truncated=true, got %#v", entry.Meta)
	}

	ttl := s.TTL(logKey)
	if ttl <= 0 {
		t.Fatalf("expected ttl set")
	}
	// expire_at + 10 minutes buffer (60 + 600 seconds).
	if ttl < 9*time.Minute || ttl > 12*time.Minute {
		t.Fatalf("unexpected ttl: %v", ttl)
	}
}

func TestWriteDeviceDebugLog_ExpiredConfigNoop(t *testing.T) {
	s, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	defer s.Close()

	redisCache = redis.NewClient(&redis.Options{Addr: s.Addr()})
	t.Cleanup(func() { _ = redisCache.Close() })

	devDebugNow = func() time.Time { return time.Unix(3000, 0) }
	t.Cleanup(func() { devDebugNow = time.Now })

	deviceID := "dev1"
	cfg := DeviceDebugConfig{
		Enabled:  true,
		ExpireAt: devDebugNow().Unix() - 1,
		MaxItems: 10,
	}
	if err := SetRedisForJsondata(devDebugCfgKey(deviceID), cfg, 0); err != nil {
		t.Fatalf("set cfg: %v", err)
	}

	wrote, err := WriteDeviceDebugLog(deviceID, DeviceDebugLogEntry{Action: "auth", Direction: "na", Outcome: "deny"})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if wrote {
		t.Fatalf("expected wrote=false for expired cfg")
	}
	got, err := s.List(devDebugLogsKey(deviceID))
	if err != nil {
		// miniredis returns an error for missing keys
		return
	}
	if len(got) != 0 {
		t.Fatalf("expected no logs, got %d", len(got))
	}
}
