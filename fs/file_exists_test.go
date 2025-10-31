package fs

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileExists(t *testing.T) {
	t.Run("returns true when file exists", func(t *testing.T) {
		// Create a temporary file
		tmpFile, err := os.CreateTemp("", "test-file-*.txt")
		require.NoError(t, err)
		defer os.Remove(tmpFile.Name())
		tmpFile.Close()

		result := FileExists(tmpFile.Name())
		assert.True(t, result)
	})

	t.Run("returns false when file does not exist", func(t *testing.T) {
		nonExistentPath := filepath.Join(os.TempDir(), "non-existent-file-12345.txt")

		result := FileExists(nonExistentPath)
		assert.False(t, result)
	})

	t.Run("returns false when path is a directory", func(t *testing.T) {
		// Create a temporary directory
		tmpDir, err := os.MkdirTemp("", "test-dir-*")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		result := FileExists(tmpDir)
		assert.False(t, result)
	})

	t.Run("returns false when path is empty string", func(t *testing.T) {
		result := FileExists("")
		assert.False(t, result)
	})

	t.Run("returns false for nested non-existent path", func(t *testing.T) {
		nonExistentPath := filepath.Join(os.TempDir(), "non-existent-dir", "non-existent-file.txt")

		result := FileExists(nonExistentPath)
		assert.False(t, result)
	})

	t.Run("handles file with special characters in name", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "test-file-with-dashes_and_underscores-*.txt")
		require.NoError(t, err)
		defer os.Remove(tmpFile.Name())
		tmpFile.Close()

		result := FileExists(tmpFile.Name())
		assert.True(t, result)
	})

	t.Run("handles file in nested directory", func(t *testing.T) {
		// Create nested directory structure
		tmpDir, err := os.MkdirTemp("", "test-nested-*")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		nestedDir := filepath.Join(tmpDir, "level1", "level2")
		err = os.MkdirAll(nestedDir, 0755)
		require.NoError(t, err)

		// Create file in nested directory
		filePath := filepath.Join(nestedDir, "test.txt")
		err = os.WriteFile(filePath, []byte("test"), 0644)
		require.NoError(t, err)

		result := FileExists(filePath)
		assert.True(t, result)
	})

	t.Run("returns true for hidden file", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", ".hidden-file-*")
		require.NoError(t, err)
		defer os.Remove(tmpFile.Name())
		tmpFile.Close()

		result := FileExists(tmpFile.Name())
		assert.True(t, result)
	})

	t.Run("returns false for relative path to non-existent file", func(t *testing.T) {
		result := FileExists("./non-existent-file.txt")
		assert.False(t, result)
	})

	t.Run("handles file with no extension", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "test-no-extension-*")
		require.NoError(t, err)
		defer os.Remove(tmpFile.Name())
		tmpFile.Close()

		result := FileExists(tmpFile.Name())
		assert.True(t, result)
	})
}
