package fs

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMkTempFile(t *testing.T) {
	t.Run("creates temp file with pattern in default temp dir", func(t *testing.T) {
		file, cleanup, err := MkTempFile("", "test-pattern-*.txt")
		require.NoError(t, err)
		require.NotNil(t, file)
		require.NotNil(t, cleanup)
		defer cleanup()

		// Verify file exists
		_, err = os.Stat(file.Name())
		assert.NoError(t, err)

		// Verify file is in system temp directory
		assert.Contains(t, file.Name(), os.TempDir())

		// Verify the file is readable and writable
		_, err = file.WriteString("test content")
		assert.NoError(t, err)

		// Seek back to the beginning
		_, err = file.Seek(0, 0)
		assert.NoError(t, err)

		buf := make([]byte, 12)
		n, err := file.Read(buf)
		assert.NoError(t, err)
		assert.Equal(t, 12, n)
		assert.Equal(t, "test content", string(buf))

		// Close the file
		err = file.Close()
		assert.NoError(t, err)
	})

	t.Run("creates temp file in specified directory", func(t *testing.T) {
		// Create a temporary directory for testing
		tempDir, err := os.MkdirTemp("", "test-dir-*")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		file, cleanup, err := MkTempFile(tempDir, "testfile-*.dat")
		require.NoError(t, err)
		require.NotNil(t, file)
		require.NotNil(t, cleanup)
		defer cleanup()

		// Verify file exists
		_, err = os.Stat(file.Name())
		assert.NoError(t, err)

		// Verify file is in the specified directory
		assert.Equal(t, tempDir, filepath.Dir(file.Name()))

		err = file.Close()
		assert.NoError(t, err)
	})

	t.Run("file has pattern-based name", func(t *testing.T) {
		pattern := "specific-name-*.txt"
		file, cleanup, err := MkTempFile("", pattern)
		require.NoError(t, err)
		require.NotNil(t, file)
		defer cleanup()
		defer file.Close()

		// Verify the file name matches the pattern (starts with prefix and ends with suffix)
		fileName := filepath.Base(file.Name())
		assert.Contains(t, fileName, "specific-name-")
		assert.Contains(t, fileName, ".txt")
	})

	t.Run("cleanup removes temp file", func(t *testing.T) {
		file, cleanup, err := MkTempFile("", "file-*.txt")
		require.NoError(t, err)
		require.NotNil(t, file)

		filePath := file.Name()

		err = file.Close()
		require.NoError(t, err)

		// Verify file exists before cleanup
		_, err = os.Stat(filePath)
		assert.NoError(t, err)

		// Call cleanup
		cleanup()

		// Verify file is removed after cleanup
		_, err = os.Stat(filePath)
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("multiple temp files are independent", func(t *testing.T) {
		file1, cleanup1, err := MkTempFile("", "file1-*.txt")
		require.NoError(t, err)
		defer cleanup1()
		defer file1.Close()

		file2, cleanup2, err := MkTempFile("", "file2-*.txt")
		require.NoError(t, err)
		defer cleanup2()
		defer file2.Close()

		// Verify files have different names
		assert.NotEqual(t, file1.Name(), file2.Name())
	})

	t.Run("file has correct permissions", func(t *testing.T) {
		file, cleanup, err := MkTempFile("", "perms-*.txt")
		require.NoError(t, err)
		defer cleanup()
		defer file.Close()

		info, err := file.Stat()
		require.NoError(t, err)

		// Check that permissions are 0o600 (default for os.CreateTemp)
		assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
	})

	t.Run("supports patterns with special characters", func(t *testing.T) {
		pattern := "test-file_123-*.tmp"
		file, cleanup, err := MkTempFile("", pattern)
		require.NoError(t, err)
		require.NotNil(t, file)
		defer cleanup()
		defer file.Close()

		fileName := filepath.Base(file.Name())
		assert.Contains(t, fileName, "test-file_123-")
		assert.Contains(t, fileName, ".tmp")
	})

	t.Run("can write and read data", func(t *testing.T) {
		file, cleanup, err := MkTempFile("", "data-*.bin")
		require.NoError(t, err)
		defer cleanup()
		defer file.Close()

		testData := []byte{0x01, 0x02, 0x03, 0x04}
		n, err := file.Write(testData)
		require.NoError(t, err)
		assert.Equal(t, len(testData), n)

		_, err = file.Seek(0, 0)
		require.NoError(t, err)

		readData := make([]byte, len(testData))
		n, err = file.Read(readData)
		require.NoError(t, err)
		assert.Equal(t, len(testData), n)
		assert.Equal(t, testData, readData)
	})

	t.Run("handles empty pattern", func(t *testing.T) {
		file, cleanup, err := MkTempFile("", "")
		require.NoError(t, err)
		require.NotNil(t, file)
		defer cleanup()
		defer file.Close()

		// Verify the file was created
		_, err = os.Stat(file.Name())
		assert.NoError(t, err)
	})
}
