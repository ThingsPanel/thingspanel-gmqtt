package util

import "testing"

func TestValidateTopic(t *testing.T) {
	var cases = []struct {
		input string
		want  bool
	}{
		{"devices/telemetry", true},
		{"devices/attributes/test", true},
		{"devices/event/test", true},
		{"gateway/attributes/test", true},
		{"gateway/event/test", true},
		{"devices/telemetry/test", false},
		{"devices/test/telemetry", false},
		{"devices_test", false},
		{"", false},
		{"xxxx/up", true},
	}

	for _, c := range cases {
		got := ValidateTopic(c.input)
		if got != c.want {
			t.Errorf("ValidateTopic(%q) == %v, want %v", c.input, got, c.want)
		}
	}
}
