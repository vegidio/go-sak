package crypto

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestXxh3Reader(t *testing.T) {
	t.Run("successful hash of simple string", func(t *testing.T) {
		reader := strings.NewReader("hello world")
		hash, err := Xxh3Reader(reader)
		require.NoError(t, err)
		// Known XXH3 hash of "hello world"
		assert.Equal(t, "df8d09e93f874900a99b8775cc15b6c7", hash)
	})

	t.Run("successful hash of empty string", func(t *testing.T) {
		reader := strings.NewReader("")
		hash, err := Xxh3Reader(reader)
		require.NoError(t, err)
		// Known XXH3 hash of empty string
		assert.Equal(t, "99aa06d3014798d86001c324468d497f", hash)
	})

	t.Run("successful hash of multiline content", func(t *testing.T) {
		content := "line1\nline2\nline3"
		reader := strings.NewReader(content)
		hash, err := Xxh3Reader(reader)
		require.NoError(t, err)
		assert.Len(t, hash, 32) // XXH3-128 produces 32 hex characters
		assert.NotEmpty(t, hash)
	})

	t.Run("hash contains only lowercase hexadecimal characters", func(t *testing.T) {
		reader := strings.NewReader("test content")
		hash, err := Xxh3Reader(reader)
		require.NoError(t, err)
		assert.Len(t, hash, 32)
		assert.Regexp(t, "^[a-f0-9]{32}$", hash)
	})

	t.Run("successful hash of large content", func(t *testing.T) {
		// Create a large string (1MB)
		largeContent := strings.Repeat("a", 1024*1024)
		reader := strings.NewReader(largeContent)
		hash, err := Xxh3Reader(reader)
		require.NoError(t, err)
		assert.Len(t, hash, 32)
		assert.NotEmpty(t, hash)
	})

	t.Run("successful hash of binary-like content", func(t *testing.T) {
		content := "\x00\x01\x02\x03\xff\xfe\xfd"
		reader := strings.NewReader(content)
		hash, err := Xxh3Reader(reader)
		require.NoError(t, err)
		assert.Len(t, hash, 32)
	})

	t.Run("same content produces same hash", func(t *testing.T) {
		content := "deterministic test"

		reader1 := strings.NewReader(content)
		hash1, err1 := Xxh3Reader(reader1)
		require.NoError(t, err1)

		reader2 := strings.NewReader(content)
		hash2, err2 := Xxh3Reader(reader2)
		require.NoError(t, err2)

		assert.Equal(t, hash1, hash2)
	})

	t.Run("different content produces different hash", func(t *testing.T) {
		reader1 := strings.NewReader("content1")
		hash1, err1 := Xxh3Reader(reader1)
		require.NoError(t, err1)

		reader2 := strings.NewReader("content2")
		hash2, err2 := Xxh3Reader(reader2)
		require.NoError(t, err2)

		assert.NotEqual(t, hash1, hash2)
	})
}

func TestXxh3File(t *testing.T) {
	t.Run("successful hash of file with content", func(t *testing.T) {
		tempFile := createTempFileXxh3(t, "hello world")
		defer os.Remove(tempFile)

		hash, err := Xxh3File(tempFile)
		require.NoError(t, err)
		// Known XXH3 hash of "hello world"
		assert.Equal(t, "df8d09e93f874900a99b8775cc15b6c7", hash)
	})

	t.Run("successful hash of empty file", func(t *testing.T) {
		tempFile := createTempFileXxh3(t, "")
		defer os.Remove(tempFile)

		hash, err := Xxh3File(tempFile)
		require.NoError(t, err)
		// Known XXH3 hash of empty string
		assert.Equal(t, "99aa06d3014798d86001c324468d497f", hash)
	})

	t.Run("error when file does not exist", func(t *testing.T) {
		hash, err := Xxh3File("/nonexistent/path/to/file.txt")
		assert.Error(t, err)
		assert.Empty(t, hash)
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("error when path is a directory", func(t *testing.T) {
		tempDir := t.TempDir()

		hash, err := Xxh3File(tempDir)
		assert.Error(t, err)
		assert.Empty(t, hash)
	})

	t.Run("successful hash of file with multiline content", func(t *testing.T) {
		content := "line1\nline2\nline3\n"
		tempFile := createTempFileXxh3(t, content)
		defer os.Remove(tempFile)

		hash, err := Xxh3File(tempFile)
		require.NoError(t, err)
		assert.Len(t, hash, 32)
		assert.NotEmpty(t, hash)
	})

	t.Run("successful hash of large file", func(t *testing.T) {
		// Create a large file (1MB)
		largeContent := strings.Repeat("test", 256*1024)
		tempFile := createTempFileXxh3(t, largeContent)
		defer os.Remove(tempFile)

		hash, err := Xxh3File(tempFile)
		require.NoError(t, err)
		assert.Len(t, hash, 32)
	})

	t.Run("same file content produces same hash", func(t *testing.T) {
		content := "consistent content"

		tempFile1 := createTempFileXxh3(t, content)
		defer os.Remove(tempFile1)

		tempFile2 := createTempFileXxh3(t, content)
		defer os.Remove(tempFile2)

		hash1, err1 := Xxh3File(tempFile1)
		require.NoError(t, err1)

		hash2, err2 := Xxh3File(tempFile2)
		require.NoError(t, err2)

		assert.Equal(t, hash1, hash2)
	})

	t.Run("different file content produces different hash", func(t *testing.T) {
		tempFile1 := createTempFileXxh3(t, "content1")
		defer os.Remove(tempFile1)

		tempFile2 := createTempFileXxh3(t, "content2")
		defer os.Remove(tempFile2)

		hash1, err1 := Xxh3File(tempFile1)
		require.NoError(t, err1)

		hash2, err2 := Xxh3File(tempFile2)
		require.NoError(t, err2)

		assert.NotEqual(t, hash1, hash2)
	})

	t.Run("successful hash of file with special characters", func(t *testing.T) {
		content := "Special: é, ñ, 中文, 🎉"
		tempFile := createTempFileXxh3(t, content)
		defer os.Remove(tempFile)

		hash, err := Xxh3File(tempFile)
		require.NoError(t, err)
		assert.Len(t, hash, 32)
	})

	t.Run("successful hash of file with binary content", func(t *testing.T) {
		content := "\x00\x01\x02\x03\xff\xfe\xfd\xfc"
		tempFile := createTempFileXxh3(t, content)
		defer os.Remove(tempFile)

		hash, err := Xxh3File(tempFile)
		require.NoError(t, err)
		assert.Len(t, hash, 32)
	})

	t.Run("error with invalid path characters", func(t *testing.T) {
		hash, err := Xxh3File("\x00invalid")
		assert.Error(t, err)
		assert.Empty(t, hash)
	})
}

func TestXxh3Reader_Xxh3File_Consistency(t *testing.T) {
	t.Run("reader and file produce same hash for same content", func(t *testing.T) {
		content := "consistency test content"

		// Hash via reader
		reader := strings.NewReader(content)
		readerHash, err := Xxh3Reader(reader)
		require.NoError(t, err)

		// Hash via file
		tempFile := createTempFileXxh3(t, content)
		defer os.Remove(tempFile)
		fileHash, err := Xxh3File(tempFile)
		require.NoError(t, err)

		assert.Equal(t, readerHash, fileHash)
	})
}

// createTempFileXxh3 creates a temporary file with the given content and returns its path
func createTempFileXxh3(t *testing.T, content string) string {
	t.Helper()

	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "testfile.txt")

	err := os.WriteFile(tempFile, []byte(content), 0644)
	require.NoError(t, err)

	return tempFile
}
