package fs

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMkTempFile(t *testing.T) {
	t.Run("creates temp file with pattern", func(t *testing.T) {
		file, cleanup, err := MkTempFile("test-pattern-*", "myfile.txt")
		require.NoError(t, err)
		require.NotNil(t, file)
		require.NotNil(t, cleanup)
		defer cleanup()

		// Verify file exists
		_, err = os.Stat(file.Name())
		assert.NoError(t, err)

		// Verify file is readable and writable
		_, err = file.WriteString("test content")
		assert.NoError(t, err)

		// Seek back to beginning
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

	t.Run("creates temp file without pattern", func(t *testing.T) {
		file, cleanup, err := MkTempFile("", "testfile.dat")
		require.NoError(t, err)
		require.NotNil(t, file)
		require.NotNil(t, cleanup)
		defer cleanup()

		// Verify file exists
		_, err = os.Stat(file.Name())
		assert.NoError(t, err)

		err = file.Close()
		assert.NoError(t, err)
	})

	t.Run("file has correct name", func(t *testing.T) {
		fileName := "specific-name.txt"
		file, cleanup, err := MkTempFile("test-*", fileName)
		require.NoError(t, err)
		require.NotNil(t, file)
		defer cleanup()
		defer file.Close()

		// Verify the file name matches
		assert.Equal(t, fileName, filepath.Base(file.Name()))
	})

	t.Run("cleanup removes temp file and directory", func(t *testing.T) {
		file, cleanup, err := MkTempFile("test-cleanup-*", "file.txt")
		require.NoError(t, err)
		require.NotNil(t, file)

		filePath := file.Name()
		dirPath := filepath.Dir(filePath)

		err = file.Close()
		require.NoError(t, err)

		// Verify file and directory exist before cleanup
		_, err = os.Stat(filePath)
		assert.NoError(t, err)
		_, err = os.Stat(dirPath)
		assert.NoError(t, err)

		// Call cleanup
		cleanup()

		// Verify file and directory are removed after cleanup
		_, err = os.Stat(filePath)
		assert.True(t, os.IsNotExist(err))
		_, err = os.Stat(dirPath)
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("multiple temp files are independent", func(t *testing.T) {
		file1, cleanup1, err := MkTempFile("test-1-*", "file1.txt")
		require.NoError(t, err)
		defer cleanup1()
		defer file1.Close()

		file2, cleanup2, err := MkTempFile("test-2-*", "file2.txt")
		require.NoError(t, err)
		defer cleanup2()
		defer file2.Close()

		// Verify files are in different directories
		assert.NotEqual(t, file1.Name(), file2.Name())
		assert.NotEqual(t, filepath.Dir(file1.Name()), filepath.Dir(file2.Name()))
	})

	t.Run("file has correct permissions", func(t *testing.T) {
		file, cleanup, err := MkTempFile("test-perms-*", "perms.txt")
		require.NoError(t, err)
		defer cleanup()
		defer file.Close()

		info, err := file.Stat()
		require.NoError(t, err)

		// Check that permissions are 0o644
		assert.Equal(t, os.FileMode(0o644), info.Mode().Perm())
	})

	t.Run("supports files with special characters in name", func(t *testing.T) {
		fileName := "test-file_123.tmp"
		file, cleanup, err := MkTempFile("test-*", fileName)
		require.NoError(t, err)
		require.NotNil(t, file)
		defer cleanup()
		defer file.Close()

		assert.Equal(t, fileName, filepath.Base(file.Name()))
	})

	t.Run("can write and read data", func(t *testing.T) {
		file, cleanup, err := MkTempFile("test-rw-*", "data.bin")
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
}
