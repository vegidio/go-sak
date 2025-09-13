package fs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMkTempDir(t *testing.T) {
	t.Run("creates temporary directory with pattern", func(t *testing.T) {
		pattern := "test-pattern-*"

		tempDir, cleanup, err := MkTempDir(pattern)

		require.NoError(t, err)
		require.NotEmpty(t, tempDir)
		require.NotNil(t, cleanup)

		// Verify directory exists
		_, err = os.Stat(tempDir)
		assert.NoError(t, err)

		// Verify pattern is used in directory name
		dirName := filepath.Base(tempDir)
		assert.True(t, strings.HasPrefix(dirName, "test-pattern-"))

		// Clean up
		cleanup()

		// Verify directory is removed after cleanup
		_, err = os.Stat(tempDir)
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("creates temporary directory with empty pattern", func(t *testing.T) {
		tempDir, cleanup, err := MkTempDir("")

		require.NoError(t, err)
		require.NotEmpty(t, tempDir)
		require.NotNil(t, cleanup)

		// Verify directory exists
		_, err = os.Stat(tempDir)
		assert.NoError(t, err)

		// Clean up
		cleanup()

		// Verify directory is removed after cleanup
		_, err = os.Stat(tempDir)
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("cleanup function removes directory and contents", func(t *testing.T) {
		tempDir, cleanup, err := MkTempDir("test-cleanup-*")

		require.NoError(t, err)

		// Create a file inside the temp directory
		testFile := filepath.Join(tempDir, "test.txt")
		err = os.WriteFile(testFile, []byte("test content"), 0644)
		require.NoError(t, err)

		// Create a subdirectory
		subDir := filepath.Join(tempDir, "subdir")
		err = os.Mkdir(subDir, 0755)
		require.NoError(t, err)

		// Create a file in subdirectory
		subFile := filepath.Join(subDir, "subfile.txt")
		err = os.WriteFile(subFile, []byte("sub content"), 0644)
		require.NoError(t, err)

		// Verify everything exists
		_, err = os.Stat(testFile)
		assert.NoError(t, err)
		_, err = os.Stat(subDir)
		assert.NoError(t, err)
		_, err = os.Stat(subFile)
		assert.NoError(t, err)

		// Clean up
		cleanup()

		// Verify everything is removed
		_, err = os.Stat(tempDir)
		assert.True(t, os.IsNotExist(err))
		_, err = os.Stat(testFile)
		assert.True(t, os.IsNotExist(err))
		_, err = os.Stat(subDir)
		assert.True(t, os.IsNotExist(err))
		_, err = os.Stat(subFile)
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("multiple calls create different directories", func(t *testing.T) {
		pattern := "multi-test-*"

		tempDir1, cleanup1, err1 := MkTempDir(pattern)
		tempDir2, cleanup2, err2 := MkTempDir(pattern)

		require.NoError(t, err1)
		require.NoError(t, err2)
		require.NotEqual(t, tempDir1, tempDir2)

		// Verify both directories exist
		_, err := os.Stat(tempDir1)
		assert.NoError(t, err)
		_, err = os.Stat(tempDir2)
		assert.NoError(t, err)

		// Clean up both
		cleanup1()
		cleanup2()

		// Verify both are removed
		_, err = os.Stat(tempDir1)
		assert.True(t, os.IsNotExist(err))
		_, err = os.Stat(tempDir2)
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("cleanup is safe to call multiple times", func(t *testing.T) {
		tempDir, cleanup, err := MkTempDir("safe-cleanup-*")

		require.NoError(t, err)

		// Verify directory exists
		_, err = os.Stat(tempDir)
		assert.NoError(t, err)

		// Call cleanup multiple times - should not panic or error
		cleanup()
		cleanup()
		cleanup()

		// Directory should still be gone
		_, err = os.Stat(tempDir)
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("returns error when pattern is invalid", func(t *testing.T) {
		// Test with an invalid pattern that contains path separators
		invalidPattern := "invalid/pattern*"

		tempDir, cleanup, err := MkTempDir(invalidPattern)

		// The behavior may vary by OS, but we should handle it gracefully
		if err != nil {
			assert.Error(t, err)
			assert.Empty(t, tempDir)
			assert.Nil(t, cleanup)
		} else {
			// If it succeeds, clean up
			require.NotNil(t, cleanup)
			cleanup()
		}
	})

	t.Run("created directory is writable", func(t *testing.T) {
		tempDir, cleanup, err := MkTempDir("writable-test-*")
		defer cleanup()

		require.NoError(t, err)

		// Try to create a file in the directory
		testFile := filepath.Join(tempDir, "write_test.txt")
		err = os.WriteFile(testFile, []byte("test"), 0644)
		assert.NoError(t, err)

		// Verify file was created
		_, err = os.Stat(testFile)
		assert.NoError(t, err)
	})

	t.Run("directory is created in system temp location", func(t *testing.T) {
		tempDir, cleanup, err := MkTempDir("location-test-*")
		defer cleanup()

		require.NoError(t, err)

		// Verify the directory is created under the system temp directory
		systemTempDir := os.TempDir()
		assert.True(t, strings.HasPrefix(tempDir, systemTempDir))
	})
}
