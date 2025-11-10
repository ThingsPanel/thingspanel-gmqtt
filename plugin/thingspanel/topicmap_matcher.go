package thingspanel

import (
	"regexp"
	"strings"
)

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


