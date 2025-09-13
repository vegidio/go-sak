package fs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMkUserConfigFile(t *testing.T) {
	t.Run("successful file creation with single directory", func(t *testing.T) {
		// Clean up after test
		defer cleanupTestConfig(t, "test-app")

		file, err := MkUserConfigFile("test-app", "config.json")
		require.NoError(t, err)
		require.NotNil(t, file)
		defer file.Close()

		// Verify file was created and is accessible
		stat, err := file.Stat()
		require.NoError(t, err)
		assert.Equal(t, "config.json", stat.Name())
		assert.Equal(t, os.FileMode(0o644), stat.Mode().Perm())

		// Verify path structure
		configDir, err := os.UserConfigDir()
		require.NoError(t, err)
		expectedPath := filepath.Join(configDir, "test-app", "config.json")
		assert.Equal(t, expectedPath, file.Name())
	})

	t.Run("successful file creation with nested directories", func(t *testing.T) {
		defer cleanupTestConfig(t, "test-app")

		file, err := MkUserConfigFile("test-app", "subdir", "nested", "config.yaml")
		require.NoError(t, err)
		require.NotNil(t, file)
		defer file.Close()

		// Verify file was created
		stat, err := file.Stat()
		require.NoError(t, err)
		assert.Equal(t, "config.yaml", stat.Name())

		// Verify nested path structure
		configDir, err := os.UserConfigDir()
		require.NoError(t, err)
		expectedPath := filepath.Join(configDir, "test-app", "subdir", "nested", "config.yaml")
		assert.Equal(t, expectedPath, file.Name())
	})

	t.Run("file already exists - should open existing file", func(t *testing.T) {
		defer cleanupTestConfig(t, "test-app")

		// Create file first time
		file1, err := MkUserConfigFile("test-app", "existing.txt")
		require.NoError(t, err)
		defer file1.Close()

		// Write some content
		content := "test content"
		_, err = file1.WriteString(content)
		require.NoError(t, err)
		file1.Close()

		// Open same file again
		file2, err := MkUserConfigFile("test-app", "existing.txt")
		require.NoError(t, err)
		defer file2.Close()

		// Verify it's the same file with existing content
		buf := make([]byte, len(content))
		_, err = file2.Read(buf)
		require.NoError(t, err)
		assert.Equal(t, content, string(buf))
	})

	t.Run("error when name is empty", func(t *testing.T) {
		file, err := MkUserConfigFile("", "config.json")
		assert.Error(t, err)
		assert.Nil(t, file)
		assert.Contains(t, err.Error(), "name cannot be empty")
	})

	t.Run("error when no path components provided", func(t *testing.T) {
		file, err := MkUserConfigFile("test-app")
		assert.Error(t, err)
		assert.Nil(t, file)
		assert.Contains(t, err.Error(), "no path components provided")
	})

	t.Run("handles special characters in name and paths", func(t *testing.T) {
		defer cleanupTestConfig(t, "test-app-with-dashes")

		file, err := MkUserConfigFile("test-app-with-dashes", "sub_dir", "config-file.json")
		require.NoError(t, err)
		require.NotNil(t, file)
		defer file.Close()

		stat, err := file.Stat()
		require.NoError(t, err)
		assert.Equal(t, "config-file.json", stat.Name())
	})

	t.Run("creates intermediate directories", func(t *testing.T) {
		defer cleanupTestConfig(t, "test-app")

		file, err := MkUserConfigFile("test-app", "level1", "level2", "level3", "deep.conf")
		require.NoError(t, err)
		require.NotNil(t, file)
		defer file.Close()

		// Verify all intermediate directories were created
		configDir, err := os.UserConfigDir()
		require.NoError(t, err)

		level1Path := filepath.Join(configDir, "test-app", "level1")
		level2Path := filepath.Join(level1Path, "level2")
		level3Path := filepath.Join(level2Path, "level3")

		assert.DirExists(t, level1Path)
		assert.DirExists(t, level2Path)
		assert.DirExists(t, level3Path)
		assert.FileExists(t, filepath.Join(level3Path, "deep.conf"))
	})

	t.Run("directory permissions are correct", func(t *testing.T) {
		defer cleanupTestConfig(t, "test-app")

		file, err := MkUserConfigFile("test-app", "permissions-test", "config.txt")
		require.NoError(t, err)
		defer file.Close()

		// Check directory permissions
		configDir, err := os.UserConfigDir()
		require.NoError(t, err)

		dirPath := filepath.Join(configDir, "test-app", "permissions-test")
		stat, err := os.Stat(dirPath)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o755), stat.Mode().Perm())
	})

	t.Run("handles whitespace in paths", func(t *testing.T) {
		defer cleanupTestConfig(t, "test app")

		file, err := MkUserConfigFile("test app", "sub dir", "config file.json")
		require.NoError(t, err)
		require.NotNil(t, file)
		defer file.Close()

		stat, err := file.Stat()
		require.NoError(t, err)
		assert.Equal(t, "config file.json", stat.Name())
	})
}

// cleanupTestConfig removes test configuration directories after tests
func cleanupTestConfig(t *testing.T, appName string) {
	t.Helper()
	configDir, err := os.UserConfigDir()
	if err != nil {
		t.Logf("Warning: could not get user config dir for cleanup: %v", err)
		return
	}

	testDir := filepath.Join(configDir, appName)
	if err := os.RemoveAll(testDir); err != nil {
		t.Logf("Warning: could not clean up test directory %s: %v", testDir, err)
	}
}

func TestMkUserConfigFile_EdgeCases(t *testing.T) {
	t.Run("single character names and paths", func(t *testing.T) {
		defer cleanupTestConfig(t, "a")

		file, err := MkUserConfigFile("a", "b", "c")
		require.NoError(t, err)
		require.NotNil(t, file)
		defer file.Close()

		stat, err := file.Stat()
		require.NoError(t, err)
		assert.Equal(t, "c", stat.Name())
	})

	t.Run("very long path components", func(t *testing.T) {
		longName := strings.Repeat("a", 100)
		defer cleanupTestConfig(t, longName)

		longPath := strings.Repeat("b", 100)
		longFileName := strings.Repeat("c", 100) + ".txt"

		file, err := MkUserConfigFile(longName, longPath, longFileName)
		require.NoError(t, err)
		require.NotNil(t, file)
		defer file.Close()

		stat, err := file.Stat()
		require.NoError(t, err)
		assert.Equal(t, longFileName, stat.Name())
	})

	t.Run("file name with no extension", func(t *testing.T) {
		defer cleanupTestConfig(t, "test-app")

		file, err := MkUserConfigFile("test-app", "config")
		require.NoError(t, err)
		require.NotNil(t, file)
		defer file.Close()

		stat, err := file.Stat()
		require.NoError(t, err)
		assert.Equal(t, "config", stat.Name())
	})
}
