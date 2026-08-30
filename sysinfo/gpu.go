package sysinfo

import (
	"errors"
	"regexp"
	"runtime"
)

// GPU detection is split by platform: gpu_darwin_probes.go, gpu_linux_probes.go and gpu_windows_probes.go hold the probes and
// parsers for each, and gpu_shared.go the helpers they have in common.
//
// None of those files carries a build tag, nor a bare _GOOS filename suffix that would imply one. The parsers and mergers are pure functions over captured
// command output, and the tests exercise every platform's parsers from whichever host they run on.

type GPUInfo struct {
	Name   string
	Vendor string
	Memory uint
}

// pciIDToken matches the "[vendor:device]" token that `lspci -nn` appends to a device description.
var macVRAMToken = regexp.MustCompile(`(?i)([0-9]+)\s*(GB|MB)`)

var pciIDToken = regexp.MustCompile(`\s*\[[0-9a-fA-F]{4}:[0-9a-fA-F]{4}]`)

// GetGPUInfo returns GPU info across macOS, Linux and Windows.
//
// Backends (best-effort):
//   - macOS: system_profiler SPDisplaysDataType (text parsing)
//   - Windows: PowerShell CIM Win32_VideoController
//   - Linux: the results of /sys/class/drm/* (VRAM when available), lspci (readable names) and nvidia-smi
//     (authoritative NVIDIA name/VRAM) are merged by PCI address, because no single source sees every GPU.
func GetGPUInfo() ([]GPUInfo, error) {
	switch runtime.GOOS {
	case "linux":
		return linuxGPUInfo()
	case "darwin":
		return darwinGPUInfo()
	case "windows":
		return windowsGPUInfo()
	default:
		return nil, errors.New("unsupported OS: " + runtime.GOOS)
	}
}
