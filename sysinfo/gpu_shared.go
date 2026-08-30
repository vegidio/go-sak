package sysinfo

import (
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// Helpers shared by the per-platform GPU probes.

// isExecNotFound reports whether err means the tool simply is not installed, as opposed to being installed and
// failing. run wraps the underlying error with %w, so this is an exact check rather than a search through the
// message text - which was both locale-dependent and prone to matching a tool's own "not found" output.
func isExecNotFound(err error) bool {
	return errors.Is(err, exec.ErrNotFound) || errors.Is(err, fs.ErrNotExist)
}

func nonEmptyLines(s string) []string {
	var out []string

	for _, ln := range strings.Split(strings.TrimSpace(s), "\n") {
		ln = strings.TrimSpace(ln)
		if ln != "" {
			out = append(out, ln)
		}
	}

	return out
}

func inferVendor(s string) string {
	u := strings.ToUpper(s)
	switch {
	case strings.Contains(u, "NVIDIA"):
		return "NVIDIA"
	case strings.Contains(u, "AMD"), strings.Contains(u, "ATI"), strings.Contains(u, "RADEON"):
		return "AMD"
	case strings.Contains(u, "INTEL"):
		return "Intel"
	case strings.Contains(u, "APPLE"):
		return "Apple"
	default:
		return ""
	}
}

func vendorFromPCI(vendorHex string) string {
	switch strings.ToLower(strings.TrimSpace(vendorHex)) {
	case "0x10de":
		return "NVIDIA"
	case "0x1002", "0x1022":
		return "AMD"
	case "0x8086":
		return "Intel"
	default:
		return vendorHex
	}
}

func readFirstLine(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	s := strings.TrimSpace(string(b))
	if s == "" {
		return ""
	}
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return strings.TrimSpace(s[:i])
	}
	return s
}

func readHexFile(path string) string {
	s := readFirstLine(path)
	if s == "" {
		return ""
	}
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
		return strings.ToLower(s)
	}
	return s
}

func readUint64File(path string) uint64 {
	s := readFirstLine(path)
	if s == "" {
		return 0
	}
	n, err := strconv.ParseUint(strings.TrimSpace(s), 10, 64)
	if err != nil {
		return 0
	}
	return n
}
