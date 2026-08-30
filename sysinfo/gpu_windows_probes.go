package sysinfo

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// Windows GPU detection, via nvidia-smi and the CIM Win32_VideoController class.//
// The "_probes" suffix is load-bearing: a file named gpu_darwin.go, gpu_linux.go or gpu_windows.go would pick up an
// implicit GOOS build constraint and silently drop out of the build on every other platform. These files must stay
// unconstrained, because the parsers below are pure functions over captured command output and the tests exercise
// them from whichever host they run on.

// region - Windows

func windowsGPUInfo() ([]GPUInfo, error) {
	var errs []error

	// Prefer NVIDIA if nvidia-smi exists (correct VRAM, like Linux)
	if gpus, err := viaNvidiaSMIWindows(); err == nil && len(gpus) > 0 {
		return gpus, nil
	} else if err != nil && !isExecNotFound(err) {
		errs = append(errs, fmt.Errorf("nvidia-smi: %w", err))
	}

	// Fallback: CIM for name/vendor (AdapterRAM is unreliable; don't trust it for >4GB)
	if gpus, err := viaWindowsCIMNameOnly(); err == nil && len(gpus) > 0 {
		return gpus, nil
	} else if err != nil {
		errs = append(errs, fmt.Errorf("windows CIM: %w", err))
	}

	if len(errs) == 0 {
		return nil, errors.New("failed to detect GPU")
	}

	return nil, fmt.Errorf("failed to detect GPU: %w", errors.Join(errs...))
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
