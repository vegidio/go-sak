package sysinfo

import (
	"encoding/json"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const (
	KiB = 1024
	MiB = 1024 * KiB
)

// runTimeout bounds every external probe. system_profiler and powershell can both wedge indefinitely, and without a
// deadline that wedges GetGPUInfo along with them.
const runTimeout = 15 * time.Second

// toolPaths pins the well-known system utilities to their absolute locations, per GOOS. Resolving these through $PATH
// would let a writable PATH entry run arbitrary code in the calling process. Third-party tools such as nvidia-smi and
// lspci have no fixed location and are still looked up normally.
var toolPaths = map[string]map[string]string{
	"darwin": {
		"system_profiler": "/usr/sbin/system_profiler",
		"sysctl":          "/usr/sbin/sysctl",
		"sh":              "/bin/sh",
	},
	"linux": {
		"sh": "/bin/sh",
	},
	"windows": {
		"powershell": `C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe`,
	},
}

// resolveTool maps a tool name to its absolute path when one is known for this platform and actually present. If the
// pinned location is missing - an unusual distribution layout, say - the name is returned unchanged so that the
// normal $PATH lookup still applies and the probe keeps working.
func resolveTool(name string) string {
	path, ok := toolPaths[runtime.GOOS][name]
	if !ok {
		return name
	}

	if _, err := os.Stat(path); err != nil {
		return name
	}

	return path
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
