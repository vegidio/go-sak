package sysinfo

import (
	"encoding/json"
	"strconv"
	"strings"
)

const (
	KiB = 1024
	MiB = 1024 * KiB
)

func anyToUint64(v any) (uint64, bool) {
	switch t := v.(type) {
	case float64:
		return uint64(t), true
	case int64:
		return uint64(t), true
	case json.Number:
		n, err := t.Int64()
		if err != nil {
			return 0, false
		}
		return uint64(n), true
	case string:
		u, err := strconv.ParseUint(strings.TrimSpace(t), 10, 64)
		if err != nil {
			return 0, false
		}
		return u, true
	default:
		return 0, false
	}
}
