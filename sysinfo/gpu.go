package sysinfo

import (
	"errors"
	"regexp"
	"runtime"
	"strconv"
	"strings"
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

	// ComputeCapability is the CUDA compute capability of an NVIDIA card, and the zero value for everything else.
	// Only the nvidia-smi probes can fill it in, so a machine without that tool reports every GPU as unknown.
	ComputeCapability ComputeCapability
}

// ComputeCapability is an NVIDIA GPU's CUDA compute capability - the "SM version" that decides which CUDA toolkits can
// target the card at all, and which no amount of driver updating changes. It is reported because a name cannot stand
// in for it: the GTX 1060 and the GTX 1660 differ by one digit and by a whole architecture generation, and a caller
// matching on model names has to be taught every card NVIDIA releases.
//
// The zero value means "not determined", which is a real and common answer - a non-NVIDIA GPU, or a driver older than
// 450.51.06, which is where nvidia-smi grew the compute_cap query field. Known separates that from a genuine reading,
// so a caller can decide for itself whether an unknown card should be treated as capable or not.
type ComputeCapability struct {
	Major int
	Minor int
}

// Known reports whether the capability was actually determined. NVIDIA has never shipped a compute capability below
// 1.0, so a zero Major can only mean the probe could not answer.
func (c ComputeCapability) Known() bool {
	return c.Major > 0
}

// AtLeast reports whether the capability is at least major.minor.
//
// An unknown capability is never "at least" anything: this answers "is the card known to be new enough", never "is it
// plausibly new enough". Callers that would rather assume the best on an unreadable card have to say so, because the
// two policies differ on exactly the machines that matter and the choice must not be made silently here.
func (c ComputeCapability) AtLeast(major, minor int) bool {
	if !c.Known() {
		return false
	}

	if c.Major != major {
		return c.Major > major
	}

	return c.Minor >= minor
}

// String renders the capability the way NVIDIA writes it, or "unknown" when it was not determined.
func (c ComputeCapability) String() string {
	if !c.Known() {
		return "unknown"
	}

	return strconv.Itoa(c.Major) + "." + strconv.Itoa(c.Minor)
}

// parseComputeCapability reads the "7.5" form that nvidia-smi prints for the compute_cap field, returning the zero
// value for anything it cannot make sense of - including the literal "[N/A]" the tool prints for a GPU it can see but
// cannot query.
func parseComputeCapability(s string) ComputeCapability {
	major, minor, found := strings.Cut(strings.TrimSpace(s), ".")
	if !found {
		return ComputeCapability{}
	}

	maj, err := strconv.Atoi(strings.TrimSpace(major))
	if err != nil || maj <= 0 {
		return ComputeCapability{}
	}

	mnr, err := strconv.Atoi(strings.TrimSpace(minor))
	if err != nil || mnr < 0 {
		return ComputeCapability{}
	}

	return ComputeCapability{Major: maj, Minor: mnr}
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
