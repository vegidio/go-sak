package sysinfo

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
)

type GPUInfo struct {
	Name   string
	Vendor string
	Memory uint
}

// GetGPUInfo returns GPU info across macOS, Linux and Windows.
//
// Backends (best-effort):
//   - macOS: system_profiler SPDisplaysDataType (text parsing)
//   - Windows: PowerShell CIM Win32_VideoController
//   - Linux:
//     1) NVIDIA: nvidia-smi (if present)
//     2) /sys/class/drm/* (VRAM for AMD amdgpu when available)
//     3) lspci fallback (name/vendor; memory unknown)
func GetGPUInfo() ([]GPUInfo, error) {
	var gpus []GPUInfo
	var err error
	var wg sync.WaitGroup

	wg.Go(func() {
		switch runtime.GOOS {
		case "linux":
			gpus, err = linuxGPUInfo()
		case "darwin":
			gpus, err = darwinGPUInfo()
		case "windows":
			gpus, err = windowsGPUInfo()
		default:
			gpus, err = nil, errors.New("unsupported OS: "+runtime.GOOS)
		}
	})

	wg.Wait()
	return gpus, err
}

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
	kv := strings.SplitN(strings.TrimSpace(line), ":", 2)
	if len(kv) != 2 {
		return "", "", false
	}
	return strings.TrimSpace(kv[0]), strings.TrimSpace(kv[1]), true
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
	re := regexp.MustCompile(`(?i)([0-9]+)\s*(GB|MB)`)
	m := re.FindAllStringSubmatch(s, -1)
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

// region - Linux

func linuxGPUInfo() ([]GPUInfo, error) {
	var errs []string

	// Prefer NVIDIA if available.
	if gpus, err := viaNvidiaSMILinux(); err == nil && len(gpus) > 0 {
		return gpus, nil
	} else if err != nil && !isExecNotFound(err) {
		errs = append(errs, "nvidia-smi: "+err.Error())
	}

	if gpus, err := viaLinuxDRMSysfs(); err == nil && len(gpus) > 0 {
		return gpus, nil
	} else if err != nil {
		errs = append(errs, "linux drm sysfs: "+err.Error())
	}

	if gpus, err := viaLinuxLspci(); err == nil && len(gpus) > 0 {
		return gpus, nil
	} else if err != nil {
		errs = append(errs, "linux lspci: "+err.Error())
	}

	if len(errs) == 0 {
		return nil, errors.New("failed to detect GPU")
	}

	return nil, errors.New("failed to detect GPU: " + strings.Join(errs, " | "))
}

func viaNvidiaSMILinux() ([]GPUInfo, error) {
	out, err := run("nvidia-smi", "--query-gpu=name,memory.total", "--format=csv,noheader,nounits")
	if err != nil {
		// Common WSL location if not on PATH
		if runtime.GOOS == "linux" {
			if _, statErr := os.Stat("/usr/lib/wsl/lib/nvidia-smi"); statErr == nil {
				out, err = run("/usr/lib/wsl/lib/nvidia-smi", "--query-gpu=name,memory.total", "--format=csv,noheader,nounits")
			}
		}
	}

	if err != nil {
		return nil, err
	}

	return parseNvidiaSMIOutput(out)
}

func viaLinuxDRMSysfs() ([]GPUInfo, error) {
	const drmPath = "/sys/class/drm"
	ents, err := os.ReadDir(drmPath)
	if err != nil {
		return nil, err
	}

	var gpus []GPUInfo

	for _, e := range ents {
		n := e.Name()
		if !strings.HasPrefix(n, "card") || strings.Contains(n, "-") {
			continue
		}

		devDir := drmPath + "/" + n + "/device"
		vendorID := readHexFile(devDir + "/vendor")
		deviceID := readHexFile(devDir + "/device")
		if vendorID == "" || deviceID == "" {
			continue
		}

		vendor := vendorFromPCI(vendorID)
		name := fmt.Sprintf("PCI GPU (%s %s)", vendorID, deviceID)

		memMiB := uint(0)
		if b := readUint64File(devDir + "/mem_info_vram_total"); b > 0 {
			memMiB = uint(b / MiB)
		}

		gpus = append(gpus, GPUInfo{Name: name, Vendor: vendor, Memory: memMiB})
	}

	if len(gpus) == 0 {
		return nil, errors.New("no drm sysfs GPUs found")
	}

	return gpus, nil
}

func viaLinuxLspci() ([]GPUInfo, error) {
	out, err := run("sh", "-c", "command -v lspci >/dev/null 2>&1 && lspci -nn | egrep -i 'vga|3d|display' || true")
	if err != nil {
		return nil, err
	}

	lines := nonEmptyLines(string(out))
	if len(lines) == 0 {
		return nil, errors.New("no lspci gpu lines")
	}

	var gpus []GPUInfo

	for _, line := range lines {
		parts := strings.SplitN(line, ":", 3)
		desc := line
		if len(parts) == 3 {
			desc = strings.TrimSpace(parts[2])
		}

		gpus = append(gpus, GPUInfo{
			Name:   desc,
			Vendor: inferVendor(desc),
			Memory: 0,
		})
	}

	return gpus, nil
}

// endregion

// region - Windows

func windowsGPUInfo() ([]GPUInfo, error) {
	var errs []string

	// Prefer NVIDIA if nvidia-smi exists (correct VRAM, like Linux)
	if gpus, err := viaNvidiaSMIWindows(); err == nil && len(gpus) > 0 {
		return gpus, nil
	} else if err != nil && !isExecNotFound(err) {
		errs = append(errs, "nvidia-smi: "+err.Error())
	}

	// Fallback: CIM for name/vendor (AdapterRAM is unreliable; don't trust it for >4GB)
	if gpus, err := viaWindowsCIMNameOnly(); err == nil && len(gpus) > 0 {
		return gpus, nil
	} else if err != nil {
		errs = append(errs, "windows CIM: "+err.Error())
	}

	if len(errs) == 0 {
		return nil, errors.New("failed to detect GPU")
	}

	return nil, errors.New("failed to detect GPU: " + strings.Join(errs, " | "))
}

func viaNvidiaSMIWindows() ([]GPUInfo, error) {
	// Only attempt if present.
	if _, err := exec.LookPath("nvidia-smi"); err != nil {
		return nil, err
	}

	// Same query as Linux.
	out, err := run("nvidia-smi", "--query-gpu=name,memory.total", "--format=csv,noheader,nounits")
	if err != nil {
		return nil, err
	}

	return parseNvidiaSMIOutput(out)
}

// parseNvidiaSMIOutput parses the CSV output from nvidia-smi and returns GPU information.
func parseNvidiaSMIOutput(out []byte) ([]GPUInfo, error) {
	lines := nonEmptyLines(string(out))
	if len(lines) == 0 {
		return nil, errors.New("no output")
	}

	var gpus []GPUInfo

	for _, line := range lines {
		parts := strings.Split(line, ",")
		if len(parts) < 2 {
			continue
		}

		name := strings.TrimSpace(parts[0])
		memStr := strings.TrimSpace(parts[1]) // MiB

		mem64, perr := strconv.ParseUint(memStr, 10, 64)
		if perr != nil {
			continue
		}

		gpus = append(gpus, GPUInfo{Name: name, Vendor: "NVIDIA", Memory: uint(mem64)})
	}

	if len(gpus) == 0 {
		return nil, errors.New("could not parse nvidia-smi output")
	}

	return gpus, nil
}

func viaWindowsCIMNameOnly() ([]GPUInfo, error) {
	ps := strings.Join([]string{
		"$g=Get-CimInstance Win32_VideoController | Select-Object Name,AdapterCompatibility;",
		"$g | ConvertTo-Json -Depth 3",
	}, " ")

	out, err := run("powershell", "-NoProfile", "-NonInteractive", "-Command", ps)
	if err != nil {
		return nil, err
	}

	trim := bytes.TrimSpace(out)
	if len(trim) == 0 {
		return nil, errors.New("empty output")
	}

	type row struct {
		Name                 string `json:"Name"`
		AdapterCompatibility string `json:"AdapterCompatibility"`
	}

	var rows []row
	if trim[0] == '[' {
		if err = json.Unmarshal(trim, &rows); err != nil {
			return nil, err
		}
	} else {
		var single row
		if err = json.Unmarshal(trim, &single); err != nil {
			return nil, err
		}
		rows = []row{single}
	}

	var gpus []GPUInfo

	for _, r := range rows {
		name := strings.TrimSpace(r.Name)
		if name == "" {
			continue
		}

		vendor := strings.TrimSpace(r.AdapterCompatibility)
		if vendor == "" {
			vendor = inferVendor(name)
		}

		// Memory intentionally 0: AdapterRAM is frequently wrong for modern GPUs.
		gpus = append(gpus, GPUInfo{Name: name, Vendor: vendor, Memory: 0})
	}

	if len(gpus) == 0 {
		return nil, errors.New("no GPU entries found")
	}

	return gpus, nil
}

// endregion

func isExecNotFound(err error) bool {
	// Covers typical Go exec errors: "executable file not found in $PATH"
	// and OS-specific variants.
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "executable file not found") ||
		strings.Contains(s, "not found") && strings.Contains(s, "nvidia-smi")
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
