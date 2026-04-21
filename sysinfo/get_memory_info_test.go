package sysinfo

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetMemoryInfo(t *testing.T) {
	switch runtime.GOOS {
	case "linux", "darwin", "windows":
	default:
		t.Skipf("unsupported OS for this test: %s", runtime.GOOS)
	}

	info, err := GetMemoryInfo()
	require.NoError(t, err)

	assert.Greater(t, info.Total, uint64(0), "total memory must be positive")
}
