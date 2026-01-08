package os

import (
	goos "os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAppendEnvPath(t *testing.T) {
	t.Run("appends path to existing PATH", func(t *testing.T) {
		// Save the original PATH and restore after the test
		originalPath := goos.Getenv("PATH")
		defer goos.Setenv("PATH", originalPath)

		// Set a known PATH value
		goos.Setenv("PATH", "/usr/bin:/bin")

		// Append a new path
		AppendEnvPath("PATH", "/custom/path")

		// Verify the new PATH
		newPath := goos.Getenv("PATH")
		expected := "/usr/bin:/bin" + string(goos.PathListSeparator) + "/custom/path"
		assert.Equal(t, expected, newPath)
	})

	t.Run("appends path to empty PATH", func(t *testing.T) {
		originalPath := goos.Getenv("PATH")
		defer goos.Setenv("PATH", originalPath)

		// Set empty PATH
		goos.Setenv("PATH", "")

		// Append a new path
		AppendEnvPath("PATH", "/new/path")

		// Verify the new PATH
		newPath := goos.Getenv("PATH")
		expected := string(goos.PathListSeparator) + "/new/path"
		assert.Equal(t, expected, newPath)
	})

	t.Run("appends multiple paths sequentially", func(t *testing.T) {
		originalPath := goos.Getenv("PATH")
		defer goos.Setenv("PATH", originalPath)

		// Set initial PATH
		goos.Setenv("PATH", "/usr/bin")

		// Append multiple paths
		AppendEnvPath("PATH", "/path1")
		AppendEnvPath("PATH", "/path2")
		AppendEnvPath("PATH", "/path3")

		// Verify all paths are appended
		newPath := goos.Getenv("PATH")
		sep := string(goos.PathListSeparator)
		expected := "/usr/bin" + sep + "/path1" + sep + "/path2" + sep + "/path3"
		assert.Equal(t, expected, newPath)
	})

	t.Run("appends path with spaces", func(t *testing.T) {
		originalPath := goos.Getenv("PATH")
		defer goos.Setenv("PATH", originalPath)

		goos.Setenv("PATH", "/usr/bin")

		// Append a path with spaces
		AppendEnvPath("PATH", "/path with spaces")

		newPath := goos.Getenv("PATH")
		expected := "/usr/bin" + string(goos.PathListSeparator) + "/path with spaces"
		assert.Equal(t, expected, newPath)
	})

	t.Run("appends empty string path", func(t *testing.T) {
		originalPath := goos.Getenv("PATH")
		defer goos.Setenv("PATH", originalPath)

		goos.Setenv("PATH", "/usr/bin")

		// Append empty string
		AppendEnvPath("PATH", "")

		newPath := goos.Getenv("PATH")
		expected := "/usr/bin" + string(goos.PathListSeparator)
		assert.Equal(t, expected, newPath)
	})

	t.Run("uses correct OS-specific path separator", func(t *testing.T) {
		originalPath := goos.Getenv("PATH")
		defer goos.Setenv("PATH", originalPath)

		goos.Setenv("PATH", "/existing/path")

		AppendEnvPath("PATH", "/new/path")

		newPath := goos.Getenv("PATH")
		// Verify the separator is present
		assert.Contains(t, newPath, string(goos.PathListSeparator))

		// Verify structure
		parts := strings.Split(newPath, string(goos.PathListSeparator))
		require.Len(t, parts, 2)
		assert.Equal(t, "/existing/path", parts[0])
		assert.Equal(t, "/new/path", parts[1])
	})

	t.Run("appends path with special characters", func(t *testing.T) {
		originalPath := goos.Getenv("PATH")
		defer goos.Setenv("PATH", originalPath)

		goos.Setenv("PATH", "/usr/bin")

		// Append a path with special characters
		AppendEnvPath("PATH", "/path-with_special.chars@123")

		newPath := goos.Getenv("PATH")
		expected := "/usr/bin" + string(goos.PathListSeparator) + "/path-with_special.chars@123"
		assert.Equal(t, expected, newPath)
	})

	t.Run("preserves existing PATH structure", func(t *testing.T) {
		originalPath := goos.Getenv("PATH")
		defer goos.Setenv("PATH", originalPath)

		// Set PATH with multiple existing paths
		sep := string(goos.PathListSeparator)
		existingPath := "/usr/bin" + sep + "/bin" + sep + "/usr/local/bin"
		goos.Setenv("PATH", existingPath)

		AppendEnvPath("PATH", "/custom/bin")

		newPath := goos.Getenv("PATH")
		expected := existingPath + sep + "/custom/bin"
		assert.Equal(t, expected, newPath)

		// Verify all original paths are still present
		assert.Contains(t, newPath, "/usr/bin")
		assert.Contains(t, newPath, "/bin")
		assert.Contains(t, newPath, "/usr/local/bin")
		assert.Contains(t, newPath, "/custom/bin")
	})

	t.Run("appends very long path", func(t *testing.T) {
		originalPath := goos.Getenv("PATH")
		defer goos.Setenv("PATH", originalPath)

		goos.Setenv("PATH", "/usr/bin")

		// Create a very long path
		longPath := "/" + strings.Repeat("a", 500)
		AppendEnvPath("PATH", longPath)

		newPath := goos.Getenv("PATH")
		expected := "/usr/bin" + string(goos.PathListSeparator) + longPath
		assert.Equal(t, expected, newPath)
	})
}
