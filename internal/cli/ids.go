package cli

import (
	"strconv"
	"strings"
)

func parsePositiveID(flagName, raw string) (uint64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, usageErr("%s is required", flagName)
	}
	n, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return 0, usageErr("%s must be a positive base-10 integer", flagName)
	}
	if n == 0 {
		return 0, usageErr("%s must be greater than 0", flagName)
	}
	return n, nil
}

func idParam(name string, id uint64) map[string]any {
	return map[string]any{name: id}
}

func parseTaskStatus(raw string) (int, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "active", "":
		return 1, nil
	case "inactive":
		return 0, nil
	default:
		return 0, usageErr("--status must be active or inactive")
	}
}
