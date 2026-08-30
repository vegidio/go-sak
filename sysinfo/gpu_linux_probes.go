package sysinfo

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"
)

// Linux GPU detection, merging nvidia-smi, the DRM sysfs tree and lspci by PCI slot.//
// The "_probes" suffix is load-bearing: a file named gpu_darwin.go, gpu_linux.go or gpu_windows.go would pick up an
// implicit GOOS build constraint and silently drop out of the build on every other platform. These files must stay
// unconstrained, because the parsers below are pure functions over captured command output and the tests exercise
// them from whichever host they run on.

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
	var errs []error
	var gpus []linuxGPU

	// The probes are merged instead of used first-wins: none of them sees every GPU on its own. A discrete GPU whose
	// DRM driver isn't loaded (or runs with modeset off) has no /sys/class/drm entry, lspci is missing on minimal
	// systems, and nvidia-smi only knows about NVIDIA cards.
	if drm, err := viaLinuxDRMSysfs(); err == nil {
		gpus = drm
	} else {
		errs = append(errs, fmt.Errorf("linux drm sysfs: %w", err))
	}

	// lspci contributes human-readable names (and GPUs that DRM didn't enumerate).
	if lspci, err := viaLinuxLspci(); err == nil {
		gpus = mergeLinuxGPUs(gpus, lspci)
	} else {
		errs = append(errs, fmt.Errorf("linux lspci: %w", err))
	}

	// nvidia-smi is authoritative for NVIDIA cards: it reports the marketing name and the real VRAM size.
	if nvidia, err := viaNvidiaSMILinux(); err == nil {
		gpus = mergeNvidiaGPUs(gpus, nvidia)
	} else if !isExecNotFound(err) {
		errs = append(errs, fmt.Errorf("nvidia-smi: %w", err))
	}

	if len(gpus) == 0 {
		if len(errs) == 0 {
			return nil, errors.New("failed to detect GPU")
		}

		return nil, fmt.Errorf("failed to detect GPU: %w", errors.Join(errs...))
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
		if idx >= 0 && claimed[idx] {
			// Two rows normalising to the same bus id must not collapse onto one entry.
			idx = -1
		}
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
