package dun

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

const timeInputHint = "use HH:MM, 6a, 6am, 6:30p, 630p, noon, or midnight"

var (
	clock24Pattern        = regexp.MustCompile(`^([0-9]{1,2}):([0-5][0-9])(?::[0-5][0-9])?$`)
	clock12Pattern        = regexp.MustCompile(`^([0-9]{1,2})(?::([0-5][0-9]))?[[:space:]]*(a|am|p|pm)$`)
	compactClock12Pattern = regexp.MustCompile(`^([0-9]{1,2})([0-5][0-9])[[:space:]]*(a|am|p|pm)$`)
)

// parseTimeInput accepts the clock forms used in config and recurring
// schedule fields. It returns 24-hour hour/minute values.
func parseTimeInput(s string) (hour, minute int, ok bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "midnight":
		return 0, 0, true
	case "noon":
		return 12, 0, true
	}

	s = strings.ToLower(strings.TrimSpace(s))
	if match := clock24Pattern.FindStringSubmatch(s); match != nil {
		hour, _ = strconv.Atoi(match[1])
		minute, _ = strconv.Atoi(match[2])
		if hour <= 23 {
			return hour, minute, true
		}
		return 0, 0, false
	}

	match := clock12Pattern.FindStringSubmatch(s)
	compact := false
	if match == nil {
		match = compactClock12Pattern.FindStringSubmatch(s)
		compact = match != nil
	}
	if match == nil {
		return 0, 0, false
	}
	hour, _ = strconv.Atoi(match[1])
	if hour < 1 || hour > 12 {
		return 0, 0, false
	}
	if compact {
		minute, _ = strconv.Atoi(match[2])
	} else if match[2] != "" {
		minute, _ = strconv.Atoi(match[2])
	}
	if strings.HasPrefix(match[3], "p") && hour != 12 {
		hour += 12
	}
	if strings.HasPrefix(match[3], "a") && hour == 12 {
		hour = 0
	}
	return hour, minute, true
}

// canonicalTimeInput validates a time input and returns its stable 24-hour
// representation for persistence and sorting.
func canonicalTimeInput(s string) (string, error) {
	hour, minute, ok := parseTimeInput(s)
	if !ok {
		return "", fmt.Errorf("invalid time %q; %s", strings.TrimSpace(s), timeInputHint)
	}
	return fmt.Sprintf("%02d:%02d", hour, minute), nil
}

// normalizeOptionalTime accepts a blank value for optional schedule fields;
// nonblank values are validated and canonicalized.
func normalizeOptionalTime(label, s string) (string, error) {
	if strings.TrimSpace(s) == "" {
		return "", nil
	}
	value, err := canonicalTimeInput(s)
	if err != nil {
		return "", fmt.Errorf("%s: %w", label, err)
	}
	return value, nil
}

func timeSortKey(s string) string {
	if value, err := canonicalTimeInput(s); err == nil {
		return value
	}
	return strings.TrimSpace(s)
}
