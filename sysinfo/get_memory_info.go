package sysinfo

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
)

type MemoryInfo struct {
	Total uint64 // bytes
}

// GetMemoryInfo returns the machine's total physical RAM (bytes) on Windows, Linux and macOS.
//
// - Linux: parses /proc/meminfo (MemTotal)
// - macOS: uses sysctl hw.memsize
// - Windows: uses PowerShell (CIM) Win32_ComputerSystem TotalPhysicalMemory
func GetMemoryInfo() (MemoryInfo, error) {
	var mem MemoryInfo
	var err error
	var wg sync.WaitGroup

	wg.Go(func() {
		switch runtime.GOOS {
		case "linux":
			mem, err = linuxTotalMemory()
		case "darwin":
			mem, err = macTotalMemory()
		case "windows":
			mem, err = windowsTotalMemory()
		default:
			mem, err = MemoryInfo{}, errors.New("unsupported OS: "+runtime.GOOS)
		}
	})

	wg.Wait()
	return mem, err
}

func linuxTotalMemory() (MemoryInfo, error) {
	b, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return MemoryInfo{}, err
	}

	for _, line := range strings.Split(string(b), "\n") {
		if strings.HasPrefix(line, "MemTotal:") {
			fields := strings.Fields(line)

			// MemTotal: <kB> kB
			if len(fields) < 2 {
				return MemoryInfo{}, errors.New("unexpected MemTotal format")
			}

			kb, err := strconv.ParseUint(fields[1], 10, 64)
			if err != nil {
				return MemoryInfo{}, err
			}

			// /proc/meminfo reports in KiB, convert to MB
			return MemoryInfo{Total: kb * KiB / 1_000_000}, nil
		}
	}

	return MemoryInfo{}, errors.New("MemTotal not found in /proc/meminfo")
}

func macTotalMemory() (MemoryInfo, error) {
	out, err := run("sysctl", "-n", "hw.memsize")
	if err != nil {
		return MemoryInfo{}, err
	}

	n, err := strconv.ParseUint(strings.TrimSpace(string(out)), 10, 64)
	if err != nil {
		return MemoryInfo{}, err
	}

	if n == 0 {
		return MemoryInfo{}, errors.New("sysctl returned 0")
	}

	return MemoryInfo{Total: n / 1_000_000}, nil
}

func windowsTotalMemory() (MemoryInfo, error) {
	// TotalPhysicalMemory is bytes.
	// Using CIM is more modern than legacy WMI/wmic.
	ps := strings.Join([]string{
		"$cs=Get-CimInstance Win32_ComputerSystem | Select-Object -First 1 TotalPhysicalMemory;",
		"$cs | ConvertTo-Json -Depth 2",
	}, " ")

	out, err := run("powershell", "-NoProfile", "-NonInteractive", "-Command", ps)
	if err != nil {
		return MemoryInfo{}, err
	}

	trim := bytes.TrimSpace(out)
	if len(trim) == 0 {
		return MemoryInfo{}, errors.New("empty PowerShell output")
	}

	type row struct {
		TotalPhysicalMemory any `json:"TotalPhysicalMemory"`
	}

	var r row
	if err = json.Unmarshal(trim, &r); err != nil {
		return MemoryInfo{}, err
	}

	total, ok := anyToUint64(r.TotalPhysicalMemory)
	if !ok || total == 0 {
		return MemoryInfo{}, errors.New("could not read TotalPhysicalMemory")
	}

	return MemoryInfo{Total: total / 1_000_000}, nil
}
