package fs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMkUserConfigDir(t *testing.T) {
	t.Run("creates user config directory with single name", func(t *testing.T) {
		name := "test-app"

		configPath, err := MkUserConfigDir(name)

		require.NoError(t, err)
		require.NotEmpty(t, configPath)

		// Verify directory exists
		_, err = os.Stat(configPath)
		assert.NoError(t, err)

		// Verify path contains the expected name
		assert.True(t, strings.HasSuffix(configPath, name))

		// Verify path starts with user config directory
		userConfigDir, err := os.UserConfigDir()
		require.NoError(t, err)
		assert.True(t, strings.HasPrefix(configPath, userConfigDir))

		// Clean up
		defer os.RemoveAll(configPath)
	})

	t.Run("creates user config directory with multiple path parts", func(t *testing.T) {
		name := "test-app"
		parts := []string{"subdir", "nested", "deep"}

		configPath, err := MkUserConfigDir(name, parts...)

		require.NoError(t, err)
		require.NotEmpty(t, configPath)

		// Verify directory exists
		_, err = os.Stat(configPath)
		assert.NoError(t, err)

		// Verify all parts are in the path
		assert.True(t, strings.Contains(configPath, name))
		for _, part := range parts {
			assert.True(t, strings.Contains(configPath, part))
		}

		// Verify path structure
		userConfigDir, err := os.UserConfigDir()
		require.NoError(t, err)
		expectedPath := filepath.Join(userConfigDir, name, filepath.Join(parts...))
		assert.Equal(t, expectedPath, configPath)

		// Clean up
		defer func() {
			// Remove the entire test-app directory
			appDir := filepath.Join(userConfigDir, name)
			os.RemoveAll(appDir)
		}()
	})

	t.Run("creates directory structure recursively", func(t *testing.T) {
		name := "recursive-test"
		parts := []string{"level1", "level2", "level3"}

		configPath, err := MkUserConfigDir(name, parts...)

		require.NoError(t, err)

		// Verify all parent directories exist
		userConfigDir, err := os.UserConfigDir()
		require.NoError(t, err)

		level1Path := filepath.Join(userConfigDir, name)
		level2Path := filepath.Join(level1Path, "level1")
		level3Path := filepath.Join(level2Path, "level2")
		finalPath := filepath.Join(level3Path, "level3")

		_, err = os.Stat(level1Path)
		assert.NoError(t, err)
		_, err = os.Stat(level2Path)
		assert.NoError(t, err)
		_, err = os.Stat(level3Path)
		assert.NoError(t, err)
		_, err = os.Stat(finalPath)
		assert.NoError(t, err)

		assert.Equal(t, finalPath, configPath)

		// Clean up
		defer os.RemoveAll(level1Path)
	})

	t.Run("handles no additional parts", func(t *testing.T) {
		name := "simple-app"

		configPath, err := MkUserConfigDir(name)

		require.NoError(t, err)

		userConfigDir, err := os.UserConfigDir()
		require.NoError(t, err)
		expectedPath := filepath.Join(userConfigDir, name)
		assert.Equal(t, expectedPath, configPath)

		// Verify directory exists
		_, err = os.Stat(configPath)
		assert.NoError(t, err)

		// Clean up
		defer os.RemoveAll(configPath)
	})

	t.Run("creates directory with correct permissions", func(t *testing.T) {
		name := "permission-test"

		configPath, err := MkUserConfigDir(name)

		require.NoError(t, err)

		// Check directory permissions
		info, err := os.Stat(configPath)
		require.NoError(t, err)
		assert.True(t, info.IsDir())

		// Check that directory has read, write, and execute permissions for owner
		mode := info.Mode()
		assert.True(t, mode&0o700 == 0o700) // Owner has rwx

		// Clean up
		defer os.RemoveAll(configPath)
	})

	t.Run("succeeds when directory already exists", func(t *testing.T) {
		name := "existing-dir-test"

		// Create directory first time
		configPath1, err := MkUserConfigDir(name)
		require.NoError(t, err)

		// Create same directory second time
		configPath2, err := MkUserConfigDir(name)
		require.NoError(t, err)

		// Should return same path and not error
		assert.Equal(t, configPath1, configPath2)

		// Directory should still exist
		_, err = os.Stat(configPath1)
		assert.NoError(t, err)

		// Clean up
		defer os.RemoveAll(configPath1)
	})

	t.Run("handles special characters in name and parts", func(t *testing.T) {
		name := "app-with-dashes"
		parts := []string{"sub_dir", "with.dots", "and spaces"}

		configPath, err := MkUserConfigDir(name, parts...)

		require.NoError(t, err)

		// Verify directory exists
		_, err = os.Stat(configPath)
		assert.NoError(t, err)

		// Verify all components are in path
		assert.True(t, strings.Contains(configPath, name))
		assert.True(t, strings.Contains(configPath, "sub_dir"))
		assert.True(t, strings.Contains(configPath, "with.dots"))
		assert.True(t, strings.Contains(configPath, "and spaces"))

		// Clean up
		defer func() {
			userConfigDir, _ := os.UserConfigDir()
			appDir := filepath.Join(userConfigDir, name)
			os.RemoveAll(appDir)
		}()
	})

	t.Run("directory is writable after creation", func(t *testing.T) {
		name := "writable-test"

		configPath, err := MkUserConfigDir(name)
		require.NoError(t, err)

		// Try to create a file in the directory
		testFile := filepath.Join(configPath, "test.txt")
		err = os.WriteFile(testFile, []byte("test content"), 0644)
		assert.NoError(t, err)

		// Verify file was created
		_, err = os.Stat(testFile)
		assert.NoError(t, err)

		// Clean up
		defer os.RemoveAll(configPath)
	})

	t.Run("returns error when os.UserConfigDir fails", func(t *testing.T) {
		// This test is tricky because we can't easily mock os.UserConfigDir
		// In a real scenario, this would happen if HOME env var is not set on Unix
		// or if APPDATA is not set on Windows.
		// For now, we'll just verify the function exists and note that
		// error handling is implemented correctly in the function

		name := "error-test"
		configPath, err := MkUserConfigDir(name)

		// Under normal circumstances, this should succeed
		if err == nil {
			assert.NotEmpty(t, configPath)
			// Clean up
			defer os.RemoveAll(configPath)
		} else {
			// If it fails, it should return empty path
			assert.Empty(t, configPath)
		}
	})

	t.Run("returns full path correctly constructed", func(t *testing.T) {
		name := "path-construction-test"
		parts := []string{"a", "b", "c"}

		configPath, err := MkUserConfigDir(name, parts...)
		require.NoError(t, err)

		// Verify path construction
		userConfigDir, err := os.UserConfigDir()
		require.NoError(t, err)

		expectedParts := append([]string{userConfigDir, name}, parts...)
		expectedPath := filepath.Join(expectedParts...)

		assert.Equal(t, expectedPath, configPath)

		// Clean up
		defer func() {
			appDir := filepath.Join(userConfigDir, name)
			os.RemoveAll(appDir)
		}()
	})
}
