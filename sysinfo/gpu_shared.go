package sysinfo

import (
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// Helpers shared by the per-platform GPU probes.

// The nvidia-smi query used by both the Linux and the Windows probe, in two forms. The columns are positional, so
// the optional one goes last and parseNvidiaSMIRows can tell the two apart by counting fields.
//
// pci.bus_id is asked for on both platforms even though only Linux merges on it, so that one parser can serve both
// without the column order depending on who is calling.
const (
	nvidiaSMIBaseFields = "--query-gpu=name,memory.total,pci.bus_id"
	nvidiaSMIFullFields = nvidiaSMIBaseFields + ",compute_cap"
	nvidiaSMIFormat     = "--format=csv,noheader,nounits"
)

// runNvidiaSMI queries nvidia-smi at bin and parses its rows, asking for the compute capability and retrying without
// it when the driver does not recognise the field.
//
// The retry is the whole point of this helper. nvidia-smi rejects the entire query when one field is unknown, and
// compute_cap only exists from the 450.51.06 driver onwards - so folding it into the base query would turn "this
// driver cannot report a compute capability" into "this machine has no GPU", on precisely the legacy cards whose
// capability a caller is most likely trying to establish. The common case still costs one process: the fallback only
// runs after a failure.
//
// A missing nvidia-smi is returned unwrapped on the first attempt rather than retried, so isExecNotFound keeps
// working for callers that treat "not installed" as unremarkable.
func runNvidiaSMI(bin string) ([]nvidiaGPU, error) {
	out, err := run(bin, nvidiaSMIFullFields, nvidiaSMIFormat)
	if err != nil {
		if isExecNotFound(err) {
			return nil, err
		}

		if out, err = run(bin, nvidiaSMIBaseFields, nvidiaSMIFormat); err != nil {
			return nil, err
		}
	}

	return parseNvidiaSMIRows(out)
}

// isExecNotFound reports whether err means the tool simply is not installed, as opposed to being installed and
// failing. run wraps the underlying error with %w, so this is an exact check rather than a search through the
// message text - which was both locale-dependent and prone to matching a tool's own "not found" output.
func isExecNotFound(err error) bool {
	return errors.Is(err, exec.ErrNotFound) || errors.Is(err, fs.ErrNotExist)
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
