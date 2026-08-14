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

// pciIDToken matches the "[vendor:device]" token that `lspci -nn` appends to a device description.
var pciIDToken = regexp.MustCompile(`\s*\[[0-9a-fA-F]{4}:[0-9a-fA-F]{4}]`)

// GetGPUInfo returns GPU info across macOS, Linux and Windows.
//
// Backends (best-effort):
//   - macOS: system_profiler SPDisplaysDataType (text parsing)
//   - Windows: PowerShell CIM Win32_VideoController
//   - Linux: the results of /sys/class/drm/* (VRAM when available), lspci (readable names) and nvidia-smi
//     (authoritative NVIDIA name/VRAM) are merged by PCI address, because no single source sees every GPU.
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

// linuxGPU is a GPU detected by one of the Linux probes, carrying the PCI address used to merge the probes.
type linuxGPU struct {
	GPUInfo
	slot string // normalized PCI address, e.g. "0000:01:00.0"; empty when the probe doesn't report one
}

// nvidiaGPU is a row of nvidia-smi output; busID is empty when the driver didn't report it.
type nvidiaGPU struct {
	name   string
	memory uint
	busID  string
}

func linuxGPUInfo() ([]GPUInfo, error) {
	var errs []string
	var gpus []linuxGPU

	// The probes are merged instead of used first-wins: none of them sees every GPU on its own. A discrete GPU whose
	// DRM driver isn't loaded (or runs with modeset off) has no /sys/class/drm entry, lspci is missing on minimal
	// systems, and nvidia-smi only knows about NVIDIA cards.
	if drm, err := viaLinuxDRMSysfs(); err == nil {
		gpus = drm
	} else {
		errs = append(errs, "linux drm sysfs: "+err.Error())
	}

	// lspci contributes human-readable names (and GPUs that DRM didn't enumerate).
	if lspci, err := viaLinuxLspci(); err == nil {
		gpus = mergeLinuxGPUs(gpus, lspci)
	} else {
		errs = append(errs, "linux lspci: "+err.Error())
	}

	// nvidia-smi is authoritative for NVIDIA cards: it reports the marketing name and the real VRAM size.
	if nvidia, err := viaNvidiaSMILinux(); err == nil {
		gpus = mergeNvidiaGPUs(gpus, nvidia)
	} else if !isExecNotFound(err) {
		errs = append(errs, "nvidia-smi: "+err.Error())
	}

	if len(gpus) == 0 {
		if len(errs) == 0 {
			return nil, errors.New("failed to detect GPU")
		}

		return nil, errors.New("failed to detect GPU: " + strings.Join(errs, " | "))
	}

	infos := make([]GPUInfo, 0, len(gpus))
	for _, gpu := range gpus {
		infos = append(infos, gpu.GPUInfo)
	}

	return infos, nil
}

func viaNvidiaSMILinux() ([]nvidiaGPU, error) {
	const query = "--query-gpu=name,memory.total,pci.bus_id"

	out, err := run("nvidia-smi", query, "--format=csv,noheader,nounits")
	if err != nil {
		// Common WSL location if not on PATH
		if runtime.GOOS == "linux" {
			if _, statErr := os.Stat("/usr/lib/wsl/lib/nvidia-smi"); statErr == nil {
				out, err = run("/usr/lib/wsl/lib/nvidia-smi", query, "--format=csv,noheader,nounits")
			}
		}
	}

	if err != nil {
		return nil, err
	}

	return parseNvidiaSMIRows(out)
}

func viaLinuxDRMSysfs() ([]linuxGPU, error) {
	const drmPath = "/sys/class/drm"
	ents, err := os.ReadDir(drmPath)
	if err != nil {
		return nil, err
	}

	var gpus []linuxGPU

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

		// The `device` entry is a symlink to the PCI device, e.g. "../../../0000:01:00.0".
		slot := ""
		if target, linkErr := os.Readlink(devDir); linkErr == nil {
			slot = normalizePCISlot(target)
		}

		gpus = append(gpus, linuxGPU{
			GPUInfo: GPUInfo{Name: name, Vendor: vendor, Memory: memMiB},
			slot:    slot,
		})
	}

	if len(gpus) == 0 {
		return nil, errors.New("no drm sysfs GPUs found")
	}

	return gpus, nil
}

func viaLinuxLspci() ([]linuxGPU, error) {
	out, err := run("sh", "-c", "command -v lspci >/dev/null 2>&1 && lspci -Dnn | egrep -i 'vga|3d|display' || true")
	if err != nil {
		return nil, err
	}

	lines := nonEmptyLines(string(out))
	if len(lines) == 0 {
		return nil, errors.New("no lspci gpu lines")
	}

	var gpus []linuxGPU

	for _, line := range lines {
		slot, desc := parseLspciLine(line)
		gpus = append(gpus, linuxGPU{
			GPUInfo: GPUInfo{Name: desc, Vendor: inferVendor(desc), Memory: 0},
			slot:    slot,
		})
	}

	return gpus, nil
}

// parseLspciLine splits a `lspci -Dnn` line into its PCI address and the device description.
//
// Example input:
//
//	0000:01:00.0 VGA compatible controller [0300]: NVIDIA Corporation GA102 [GeForce RTX 3090] [10de:2204] (rev a1)
//
// which yields the slot "0000:01:00.0" and the description "NVIDIA Corporation GA102 [GeForce RTX 3090] (rev a1)".
func parseLspciLine(line string) (string, string) {
	line = strings.TrimSpace(line)

	slot, rest, found := strings.Cut(line, " ")
	if !found {
		return "", line
	}

	// Everything after the device class (e.g. "VGA compatible controller [0300]: ") is the description.
	desc := rest
	if _, after, ok := strings.Cut(rest, ": "); ok {
		desc = after
	}

	// Drop the "[vendor:device]" ID token; it's noise in a display name and the vendor is inferred from the text.
	desc = pciIDToken.ReplaceAllString(desc, "")
	desc = strings.Join(strings.Fields(desc), " ")

	return normalizePCISlot(slot), desc
}

// mergeLinuxGPUs adds extra GPUs to base, merging entries that share a PCI address instead of duplicating them.
func mergeLinuxGPUs(base, extra []linuxGPU) []linuxGPU {
	for _, gpu := range extra {
		idx := indexBySlot(base, gpu.slot)
		if idx < 0 {
			base = append(base, gpu)
			continue
		}

		// A readable name ("NVIDIA Corporation GA102 ...") beats the synthetic PCI-ID one from sysfs.
		if gpu.Name != "" {
			base[idx].Name = gpu.Name
		}
		if isUnknownVendor(base[idx].Vendor) && gpu.Vendor != "" {
			base[idx].Vendor = gpu.Vendor
		}
		if base[idx].Memory == 0 {
			base[idx].Memory = gpu.Memory
		}
	}

	return base
}

// mergeNvidiaGPUs folds nvidia-smi rows into the GPUs found by the PCI probes, appending the ones they missed. Rows are
// matched by PCI address when the driver reports one, falling back to the order in which NVIDIA GPUs were detected.
func mergeNvidiaGPUs(base []linuxGPU, nvidia []nvidiaGPU) []linuxGPU {
	claimed := make(map[int]bool, len(nvidia))

	for _, gpu := range nvidia {
		idx := indexBySlot(base, gpu.busID)
		if idx < 0 {
			idx = -1
			for i := range base {
				if strings.EqualFold(base[i].Vendor, "NVIDIA") && !claimed[i] {
					idx = i
					break
				}
			}
		}

		if idx < 0 {
			base = append(base, linuxGPU{
				GPUInfo: GPUInfo{Name: gpu.name, Vendor: "NVIDIA", Memory: gpu.memory},
				slot:    gpu.busID,
			})
			continue
		}

		claimed[idx] = true
		base[idx].Name = gpu.name
		base[idx].Vendor = "NVIDIA"
		base[idx].Memory = gpu.memory
	}

	return base
}

func indexBySlot(gpus []linuxGPU, slot string) int {
	if slot == "" {
		return -1
	}

	for i := range gpus {
		if gpus[i].slot == slot {
			return i
		}
	}

	return -1
}

// normalizePCISlot brings the PCI addresses reported by sysfs symlinks ("../../../0000:01:00.0"), lspci ("01:00.0" or
// "0000:01:00.0") and nvidia-smi ("00000000:01:00.0") into the same "0000:01:00.0" form so they can be compared.
func normalizePCISlot(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return ""
	}

	if i := strings.LastIndexByte(s, '/'); i >= 0 {
		s = s[i+1:]
	}

	parts := strings.Split(s, ":")
	switch len(parts) {
	case 2: // bus:device.function, without domain
		return "0000:" + parts[0] + ":" + parts[1]
	case 3:
		domain := parts[0]
		if len(domain) > 4 { // nvidia-smi pads the domain to 8 hex digits
			domain = domain[len(domain)-4:]
		}
		return strings.Repeat("0", max(0, 4-len(domain))) + domain + ":" + parts[1] + ":" + parts[2]
	default:
		return s
	}
}

// isUnknownVendor reports whether a vendor string is a raw PCI ID, i.e. vendorFromPCI couldn't name it.
func isUnknownVendor(vendor string) bool {
	return vendor == "" || strings.HasPrefix(strings.ToLower(vendor), "0x")
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
	rows, err := parseNvidiaSMIRows(out)
	if err != nil {
		return nil, err
	}

	gpus := make([]GPUInfo, 0, len(rows))
	for _, row := range rows {
		gpus = append(gpus, GPUInfo{Name: row.name, Vendor: "NVIDIA", Memory: row.memory})
	}

	return gpus, nil
}

// parseNvidiaSMIRows parses the CSV output from nvidia-smi. The PCI bus ID is optional: it's only present when it was
// part of the query, and is left empty otherwise.
func parseNvidiaSMIRows(out []byte) ([]nvidiaGPU, error) {
	lines := nonEmptyLines(string(out))
	if len(lines) == 0 {
		return nil, errors.New("no output")
	}

	var gpus []nvidiaGPU

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

		busID := ""
		if len(parts) > 2 {
			busID = normalizePCISlot(parts[2])
		}

		gpus = append(gpus, nvidiaGPU{name: name, memory: uint(mem64), busID: busID})
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
