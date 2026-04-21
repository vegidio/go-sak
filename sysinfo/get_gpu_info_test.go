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
