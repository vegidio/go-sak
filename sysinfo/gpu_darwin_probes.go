package sysinfo

import (
	"errors"
	"strconv"
	"strings"
)

// macOS GPU detection, via system_profiler's text output.//
// The "_probes" suffix is load-bearing: a file named gpu_darwin.go, gpu_linux.go or gpu_windows.go would pick up an
// implicit GOOS build constraint and silently drop out of the build on every other platform. These files must stay
// unconstrained, because the parsers below are pure functions over captured command output and the tests exercise
// them from whichever host they run on.

// region - macOS

func darwinGPUInfo() ([]GPUInfo, error) {
	if gpus, err := viaMacSystemProfilerText(); err == nil && len(gpus) > 0 {
		return gpus, nil
	} else if err != nil {
		return nil, errors.New("mac system_profiler: " + err.Error())
	}

	return nil, errors.New("failed to detect GPU")
}

func viaMacSystemProfilerText() ([]GPUInfo, error) {
	// Text output is more stable across macOS versions than -json for this use case.
	out, err := run("system_profiler", "SPDisplaysDataType")
	if err != nil {
		return nil, err
	}

	gpus := parseMacGPUBlocks(string(out))
	if len(gpus) == 0 {
		return nil, errors.New("no GPU entries found")
	}

	return gpus, nil
}

func parseMacGPUBlocks(output string) []GPUInfo {
	lines := strings.Split(output, "\n")
	var gpus []GPUInfo
	var cur GPUInfo
	var inGPUBlock bool

	flushGPU := func() {
		if shouldAddGPU(cur, inGPUBlock) {
			if cur.Vendor == "" {
				cur.Vendor = inferVendor(cur.Name)
			}
			gpus = append(gpus, cur)
		}
		cur = GPUInfo{}
		inGPUBlock = false
	}

	for _, raw := range lines {
		line := strings.TrimRight(raw, "\r")

		if strings.TrimSpace(line) == "" {
			flushGPU()
			continue
		}

		key, val, ok := parseKeyValue(line)
		if !ok {
			continue
		}

		processMacGPUField(&cur, &inGPUBlock, key, val)
	}

	flushGPU()
	return gpus
}

func shouldAddGPU(gpu GPUInfo, inBlock bool) bool {
	name := strings.TrimSpace(gpu.Name)
	return inBlock && name != "" && !strings.EqualFold(name, "Color LCD")
}

func parseKeyValue(line string) (string, string, bool) {
	key, val, found := strings.Cut(strings.TrimSpace(line), ":")
	if !found {
		return "", "", false
	}

	return strings.TrimSpace(key), strings.TrimSpace(val), true
}

func processMacGPUField(cur *GPUInfo, inBlock *bool, key, val string) {
	switch key {
	case "Chipset Model":
		*inBlock = true
		cur.Name = val
	case "Vendor":
		*inBlock = true
		cur.Vendor = normalizeMacVendor(val)
	case "VRAM (Total)", "VRAM":
		*inBlock = true
		if cur.Memory == 0 {
			cur.Memory = parseMacVRAMToMiB(val)
		}
	}
}

func normalizeMacVendor(v string) string {
	// Examples: "Apple", "Intel", "AMD (0x1002)", "NVIDIA (0x10de)"
	u := strings.ToUpper(v)
	switch {
	case strings.Contains(u, "APPLE"):
		return "Apple"
	case strings.Contains(u, "INTEL"):
		return "Intel"
	case strings.Contains(u, "AMD"), strings.Contains(u, "ATI"):
		return "AMD"
	case strings.Contains(u, "NVIDIA"):
		return "NVIDIA"
	default:
		// Strip PCI suffix if present.
		if i := strings.Index(v, "("); i > 0 {
			return strings.TrimSpace(v[:i])
		}
		return strings.TrimSpace(v)
	}
}

func parseMacVRAMToMiB(s string) uint {
	// Examples: "8 GB", "1536 MB", "Intel UHD Graphics 617" (ignore), "Dynamic, Max: 1536 MB"
	// We grab the last "<number> <unit>" occurrence.
	m := macVRAMToken.FindAllStringSubmatch(s, -1)
	if len(m) == 0 {
		return 0
	}

	last := m[len(m)-1]
	n, _ := strconv.ParseUint(last[1], 10, 64)
	unit := strings.ToUpper(last[2])
	if unit == "GB" {
		return uint(n * KiB)
	}

	return uint(n)
}

// endregion
