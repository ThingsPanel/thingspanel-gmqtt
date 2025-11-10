package thingspanel

import (
	"regexp"
	"strings"
)

// tryExtractWithPattern compiles a regex from a template where {device_number} becomes a capture group.
// '+' becomes single-level matcher. Returns the first capture group if matched.
func tryExtractWithPattern(template string, actual string) (string, bool) {
	if strings.Contains(template, "#") {
		return "", false
	}
	p := template
	// capture group for device_number
	p = strings.ReplaceAll(p, "{device_number}", "([^/]+)")
	// other variables treated as non-capturing single-level
	p = regexp.MustCompile(`\{[a-zA-Z0-9_]+\}`).ReplaceAllString(p, `[^/]+`)
	// '+' wildcard
	p = strings.ReplaceAll(p, "+", `[^/]+`)
	p = "^" + p + "$"
	rx, err := regexp.Compile(p)
	if err != nil {
		return "", false
	}
	m := rx.FindStringSubmatch(actual)
	if len(m) >= 2 {
		return m[1], true
	}
	return "", false
}

// TryExtractDeviceNumberFromNormalized attempts to extract {device_number} from any known normalized downlink topic.
// The set of patterns is based on the design doc's "下行" topic table.
func TryExtractDeviceNumberFromNormalized(topic string) (string, bool) {
	patterns := []string{
		"devices/telemetry/control/{device_number}",
		"devices/attributes/set/{device_number}/+",
		"devices/attributes/get/{device_number}",
		"devices/command/{device_number}/+",
		"ota/devices/inform/{device_number}",
		"gateway/telemetry/control/{device_number}",
		"gateway/attributes/set/{device_number}/+",
		"gateway/attributes/get/{device_number}",
		"gateway/command/{device_number}/+",
		"devices/attributes/response/{device_number}/+",
		"devices/event/response/{device_number}/+",
		"gateway/attributes/response/{device_number}/+",
		"gateway/event/response/{device_number}/+",
	}
	for _, tpl := range patterns {
		if dn, ok := tryExtractWithPattern(tpl, topic); ok {
			return dn, true
		}
	}
	return "", false
}

// compileSourcePattern converts source_topic with '+' and variables into a regex.
// Rules:
// - '+' matches single level: [^/]+
// - do NOT allow '#' multi-level
// - variables like {device_number} or {message_id} treated as [^/]+ when matching
func compileSourcePattern(source string) (*regexp.Regexp, bool) {
	if strings.Contains(source, "#") {
		return nil, false
	}
	p := source
	// Replace variables with single-level matchers
	p = regexp.MustCompile(`\{[a-zA-Z0-9_]+\}`).ReplaceAllString(p, `[^/]+`)
	// Replace '+' wildcard
	p = strings.ReplaceAll(p, "+", `[^/]+`)
	// Anchor full topic
	p = "^" + p + "$"
	rx, err := regexp.Compile(p)
	if err != nil {
		return nil, false
	}
	return rx, true
}

// applyTarget renders the target topic by substituting variables from the source concrete topic if present.
// For now, variables are not extracted; keep target as-is (it may also contain variables resolved by caller).
func applyTarget(target string, _ string) string {
	return target
}

// compileTargetPattern is identical to compileSourcePattern for our purposes:
// it builds a full-match regex from a target topic pattern.
func compileTargetPattern(target string) (*regexp.Regexp, bool) {
	// Disallow '#'
	if strings.Contains(target, "#") {
		return nil, false
	}
	p := target
	p = regexp.MustCompile(`\{[a-zA-Z0-9_]+\}`).ReplaceAllString(p, `[^/]+`)
	p = strings.ReplaceAll(p, "+", `[^/]+`)
	p = "^" + p + "$"
	rx, err := regexp.Compile(p)
	if err != nil {
		return nil, false
	}
	return rx, true
}

// renderTopicFromTemplate replaces variables like {device_number} in template using vars map.
// Unknown variables are left as-is; caller should ensure the result is a concrete topic.
func renderTopicFromTemplate(template string, vars map[string]string) string {
	out := template
	for k, v := range vars {
		ph := "{" + k + "}"
		out = strings.ReplaceAll(out, ph, v)
	}
	return out
}
