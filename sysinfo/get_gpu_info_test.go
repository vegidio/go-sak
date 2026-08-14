package sysinfo

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetGPUInfo(t *testing.T) {
	switch runtime.GOOS {
	case "linux", "darwin", "windows":
	default:
		t.Skipf("unsupported OS for this test: %s", runtime.GOOS)
	}

	gpus, err := GetGPUInfo()
	if err != nil {
		t.Skipf("GPU probing failed on this host (acceptable in headless/CI): %v", err)
	}

	for i, g := range gpus {
		assert.NotEmptyf(t, g.Name, "gpu[%d] name should be populated", i)
	}
}

func TestNormalizePCISlot(t *testing.T) {
	tests := map[string]string{
		"../../../0000:01:00.0": "0000:01:00.0", // sysfs symlink target
		"01:00.0":               "0000:01:00.0", // lspci without domain
		"0000:01:00.0":          "0000:01:00.0", // lspci -D
		"00000000:01:00.0":      "0000:01:00.0", // nvidia-smi
		"00000000:0A:00.0":      "0000:0a:00.0",
		"":                      "",
	}

	for input, expected := range tests {
		assert.Equalf(t, expected, normalizePCISlot(input), "input: %q", input)
	}
}

func TestParseLspciLine(t *testing.T) {
	slot, desc := parseLspciLine(
		"0000:01:00.0 VGA compatible controller [0300]: NVIDIA Corporation GA102 [GeForce RTX 3090] [10de:2204] (rev a1)")
	assert.Equal(t, "0000:01:00.0", slot)
	assert.Equal(t, "NVIDIA Corporation GA102 [GeForce RTX 3090] (rev a1)", desc)

	slot, desc = parseLspciLine("11:00.0 Display controller [0380]: Advanced Micro Devices, Inc. [AMD/ATI] Raphael [1002:164e] (rev c9)")
	assert.Equal(t, "0000:11:00.0", slot)
	assert.Equal(t, "Advanced Micro Devices, Inc. [AMD/ATI] Raphael (rev c9)", desc)
}

// A discrete NVIDIA GPU without a DRM card entry (driver not loaded, or modeset off) must still be reported, next to
// the iGPU that sysfs did enumerate.
func TestMergeLinuxGPUsAddsGPUsMissingFromSysfs(t *testing.T) {
	drm := []linuxGPU{{
		GPUInfo: GPUInfo{Name: "PCI GPU (0x1002 0x13c0)", Vendor: "AMD", Memory: 2048},
		slot:    "0000:12:00.0",
	}}

	lspci := []linuxGPU{
		{GPUInfo: GPUInfo{Name: "Advanced Micro Devices, Inc. [AMD/ATI] Raphael", Vendor: "AMD"}, slot: "0000:12:00.0"},
		{GPUInfo: GPUInfo{Name: "NVIDIA Corporation GA102 [GeForce RTX 3090]", Vendor: "NVIDIA"}, slot: "0000:01:00.0"},
	}

	merged := mergeLinuxGPUs(drm, lspci)

	assert.Len(t, merged, 2)
	// Same PCI address: enriched with the readable name, VRAM from sysfs kept.
	assert.Equal(t, "Advanced Micro Devices, Inc. [AMD/ATI] Raphael", merged[0].Name)
	assert.Equal(t, uint(2048), merged[0].Memory)
	// Unknown to sysfs: appended instead of dropped.
	assert.Equal(t, "NVIDIA", merged[1].Vendor)
}

func TestMergeLinuxGPUsNamesUnknownVendor(t *testing.T) {
	drm := []linuxGPU{{GPUInfo: GPUInfo{Name: "PCI GPU (0x1234 0x5678)", Vendor: "0x1234"}, slot: "0000:01:00.0"}}
	lspci := []linuxGPU{{GPUInfo: GPUInfo{Name: "Intel Corporation Arc", Vendor: "Intel"}, slot: "0000:01:00.0"}}

	merged := mergeLinuxGPUs(drm, lspci)

	assert.Len(t, merged, 1)
	assert.Equal(t, "Intel", merged[0].Vendor)
}

func TestMergeNvidiaGPUsRefinesMatchingSlot(t *testing.T) {
	base := []linuxGPU{
		{GPUInfo: GPUInfo{Name: "AMD Raphael", Vendor: "AMD", Memory: 2048}, slot: "0000:12:00.0"},
		{GPUInfo: GPUInfo{Name: "NVIDIA Corporation GA102", Vendor: "NVIDIA"}, slot: "0000:01:00.0"},
	}

	merged := mergeNvidiaGPUs(base, []nvidiaGPU{{name: "NVIDIA GeForce RTX 3090", memory: 24576, busID: "0000:01:00.0"}})

	assert.Len(t, merged, 2)
	assert.Equal(t, "NVIDIA GeForce RTX 3090", merged[1].Name)
	assert.Equal(t, uint(24576), merged[1].Memory)
	assert.Equal(t, "AMD Raphael", merged[0].Name)
}

// Older drivers may not report a bus ID; the rows then fall back to matching the NVIDIA GPUs in detection order.
func TestMergeNvidiaGPUsWithoutBusID(t *testing.T) {
	base := []linuxGPU{
		{GPUInfo: GPUInfo{Name: "PCI GPU (0x10de 0x2204)", Vendor: "NVIDIA"}, slot: "0000:01:00.0"},
		{GPUInfo: GPUInfo{Name: "PCI GPU (0x10de 0x2484)", Vendor: "NVIDIA"}, slot: "0000:02:00.0"},
	}

	merged := mergeNvidiaGPUs(base, []nvidiaGPU{
		{name: "NVIDIA GeForce RTX 3090", memory: 24576},
		{name: "NVIDIA GeForce RTX 3070", memory: 8192},
	})

	assert.Len(t, merged, 2)
	assert.Equal(t, "NVIDIA GeForce RTX 3090", merged[0].Name)
	assert.Equal(t, "NVIDIA GeForce RTX 3070", merged[1].Name)
}

func TestMergeNvidiaGPUsAppendsUnknownGPU(t *testing.T) {
	base := []linuxGPU{{GPUInfo: GPUInfo{Name: "AMD Raphael", Vendor: "AMD", Memory: 2048}, slot: "0000:12:00.0"}}

	merged := mergeNvidiaGPUs(base, []nvidiaGPU{{name: "NVIDIA GeForce RTX 3090", memory: 24576, busID: "0000:01:00.0"}})

	assert.Len(t, merged, 2)
	assert.Equal(t, "NVIDIA GeForce RTX 3090", merged[1].Name)
	assert.Equal(t, uint(24576), merged[1].Memory)
}

func TestParseNvidiaSMIRows(t *testing.T) {
	rows, err := parseNvidiaSMIRows([]byte("NVIDIA GeForce RTX 3090, 24576, 00000000:01:00.0\n"))

	assert.NoError(t, err)
	assert.Len(t, rows, 1)
	assert.Equal(t, "NVIDIA GeForce RTX 3090", rows[0].name)
	assert.Equal(t, uint(24576), rows[0].memory)
	assert.Equal(t, "0000:01:00.0", rows[0].busID)

	// The bus ID is optional (the Windows probe doesn't query it).
	rows, err = parseNvidiaSMIRows([]byte("NVIDIA GeForce RTX 3090, 24576\n"))

	assert.NoError(t, err)
	assert.Len(t, rows, 1)
	assert.Empty(t, rows[0].busID)
}
