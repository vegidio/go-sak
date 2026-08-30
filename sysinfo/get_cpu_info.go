package sysinfo

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"runtime"
	"strconv"
	"strings"
)

type CPUInfo struct {
	Name  string
	Cores uint
}

// GetCPUInfo returns the CPU model/name and the number of logical cores on Linux, macOS and Windows.
func GetCPUInfo() (CPUInfo, error) {
	switch runtime.GOOS {
	case "linux":
		return linuxCPUInfo()
	case "darwin":
		return macCPUInfo()
	case "windows":
		return windowsCPUInfo()
	default:
		return CPUInfo{}, errors.New("unsupported OS: " + runtime.GOOS)
	}
}

// region - Linux

func linuxCPUInfo() (CPUInfo, error) {
	// Model name from /proc/cpuinfo; logical cores via runtime.NumCPU().
	name := parseCPUNameFromProcInfo()

	if name == "" {
		name = parseCPUNameFromLscpu()
	}

	cores := uint(runtime.NumCPU())
	if cores == 0 {
		return CPUInfo{}, errors.New("could not determine core count")
	}

	if name == "" {
		name = "Unknown CPU"
	}

	return CPUInfo{Name: name, Cores: cores}, nil
}

func parseCPUNameFromProcInfo() string {
	b, err := os.ReadFile("/proc/cpuinfo")
	if err != nil {
		return ""
	}

	// x86: "model name\t: Intel(R)..."
	// ARM: sometimes "Hardware\t: ..." or "Processor\t: ..."
	for _, line := range strings.Split(string(b), "\n") {
		if strings.HasPrefix(line, "model name") || strings.HasPrefix(line, "Processor") || strings.HasPrefix(line, "Hardware") {
			if _, val, found := strings.Cut(line, ":"); found {
				if name := strings.TrimSpace(val); name != "" {
					return name
				}
			}
		}
	}

	return ""
}

func parseCPUNameFromLscpu() string {
	out, err := run("sh", "-c", "command -v lscpu >/dev/null 2>&1 && lscpu || true")
	if err != nil {
		return ""
	}

	for _, line := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "Model name:") {
			if _, val, found := strings.Cut(line, ":"); found {
				return strings.TrimSpace(val)
			}
		}
	}

	return ""
}

// endregion

// region - macOS

func macCPUInfo() (CPUInfo, error) {
	// Name: brand string on Intel; on Apple Silicon often returns "Apple M1"/"Apple M2" etc.
	nameOut, err := run("sysctl", "-n", "machdep.cpu.brand_string")
	if err != nil {
		// Apple Silicon / some configs: brand_string may not exist; fall back.
		nameOut, err = run("sysctl", "-n", "hw.model")
		if err != nil {
			return CPUInfo{}, err
		}
	}

	name := strings.TrimSpace(string(nameOut))
	if name == "" {
		name = "Unknown CPU"
	}

	// Logical cores.
	coresOut, err := run("sysctl", "-n", "hw.logicalcpu")
	if err != nil {
		// Fallback.
		coresOut, err = run("sysctl", "-n", "hw.ncpu")
		if err != nil {
			return CPUInfo{}, err
		}
	}

	c64, err := strconv.ParseUint(strings.TrimSpace(string(coresOut)), 10, 64)
	if err != nil || c64 == 0 {
		return CPUInfo{}, errors.New("could not parse core count")
	}

	return CPUInfo{Name: name, Cores: uint(c64)}, nil
}

// endregion

// region - Windows

func windowsCPUInfo() (CPUInfo, error) {
	// Use CIM for name; logical cores via NumberOfLogicalProcessors.
	ps := strings.Join([]string{
		"$c=Get-CimInstance Win32_Processor | Select-Object -First 1 Name,NumberOfLogicalProcessors;",
		"$c | ConvertTo-Json -Depth 2",
	}, " ")

	out, err := run("powershell", "-NoProfile", "-NonInteractive", "-Command", ps)
	if err != nil {
		return CPUInfo{}, err
	}

	trim := bytes.TrimSpace(out)
	if len(trim) == 0 {
		return CPUInfo{}, errors.New("empty PowerShell output")
	}

	type row struct {
		Name                      string `json:"Name"`
		NumberOfLogicalProcessors any    `json:"NumberOfLogicalProcessors"`
	}

	var r row
	if err = json.Unmarshal(trim, &r); err != nil {
		return CPUInfo{}, err
	}

	name := strings.TrimSpace(r.Name)
	if name == "" {
		name = "Unknown CPU"
	}

	c64, ok := anyToUint64(r.NumberOfLogicalProcessors)
	if !ok || c64 == 0 {
		return CPUInfo{}, errors.New("could not read NumberOfLogicalProcessors")
	}

	return CPUInfo{Name: name, Cores: uint(c64)}, nil
}

// endregion
