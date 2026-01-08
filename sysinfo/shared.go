package sysinfo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

const (
	KiB = 1024
	MiB = 1024 * KiB
	GiB = 1024 * MiB
)

func run(name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return nil, fmt.Errorf("%s: %s", name, msg)
	}
	return out, nil
}

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
