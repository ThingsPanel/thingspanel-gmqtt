package util

import "testing"

func TestValidateSubTopic(t *testing.T) {
	var cases = []struct {
		input string
		want  bool
	}{
		{"devices/telemetry", false},
		{"devices/telemetry/xxxxxx/+", true},
		{"devices/attributes/set/xxxxxx/+", true},
		{"devices/attributes/set/+/+", false},
		{"devices/attributes/get/xxxxxx", true},
		{"devices/command/xxxxx/+", true},
		{"ota/devices/infrom/xxxxx", true},
		{"evices/attributes/response/xxxx/+", true},
		{"devices/event/response/xxxxxx/+", true},
		{"", false},
		{"001/down", true},
	}

	for _, c := range cases {
		got := ValidateSubTopic(c.input)
		if got != c.want {
			t.Errorf("ValidateSubTopic(%q) == %v, want %v", c.input, got, c.want)
		}
	}
}
