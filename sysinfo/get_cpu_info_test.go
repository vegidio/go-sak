package sysinfo

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetCPUInfo(t *testing.T) {
	switch runtime.GOOS {
	case "linux", "darwin", "windows":
		// supported; continue
	default:
		t.Skipf("unsupported OS for this test: %s", runtime.GOOS)
	}

	info, err := GetCPUInfo()
	require.NoError(t, err)

	assert.NotEmpty(t, info.Name, "CPU name should be populated")
	assert.Greater(t, info.Cores, uint(0), "core count must be positive")
}
