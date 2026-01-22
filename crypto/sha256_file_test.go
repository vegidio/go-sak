package crypto

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSha256Reader(t *testing.T) {
	t.Run("successful hash of simple string", func(t *testing.T) {
		reader := strings.NewReader("hello world")
		hash, err := Sha256Reader(reader)
		require.NoError(t, err)
		// Known SHA-256 hash of "hello world"
		assert.Equal(t, "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9", hash)
	})

	t.Run("successful hash of empty string", func(t *testing.T) {
		reader := strings.NewReader("")
		hash, err := Sha256Reader(reader)
		require.NoError(t, err)
		// Known SHA-256 hash of empty string
		assert.Equal(t, "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855", hash)
	})

	t.Run("successful hash of multiline content", func(t *testing.T) {
		content := "line1\nline2\nline3"
		reader := strings.NewReader(content)
		hash, err := Sha256Reader(reader)
		require.NoError(t, err)
		// Known SHA-256 hash of "line1\nline2\nline3"
		assert.Equal(t, "6bb6a5ad9b9c43a7cb535e636578716b64ac42edea814a4cad102ba404946837", hash)
		assert.Len(t, hash, 64) // SHA-256 produces 64 hex characters
	})

	t.Run("hash contains only lowercase hexadecimal characters", func(t *testing.T) {
		reader := strings.NewReader("test content")
		hash, err := Sha256Reader(reader)
		require.NoError(t, err)
		assert.Len(t, hash, 64)
		assert.Regexp(t, "^[a-f0-9]{64}$", hash)
	})

	t.Run("successful hash of large content", func(t *testing.T) {
		// Create a large string (1MB)
		largeContent := strings.Repeat("a", 1024*1024)
		reader := strings.NewReader(largeContent)
		hash, err := Sha256Reader(reader)
		require.NoError(t, err)
		assert.Len(t, hash, 64)
		assert.NotEmpty(t, hash)
	})

	t.Run("successful hash of binary-like content", func(t *testing.T) {
		content := "\x00\x01\x02\x03\xff\xfe\xfd"
		reader := strings.NewReader(content)
		hash, err := Sha256Reader(reader)
		require.NoError(t, err)
		assert.Len(t, hash, 64)
	})

	t.Run("same content produces same hash", func(t *testing.T) {
		content := "deterministic test"

		reader1 := strings.NewReader(content)
		hash1, err1 := Sha256Reader(reader1)
		require.NoError(t, err1)

		reader2 := strings.NewReader(content)
		hash2, err2 := Sha256Reader(reader2)
		require.NoError(t, err2)

		assert.Equal(t, hash1, hash2)
	})

	t.Run("different content produces different hash", func(t *testing.T) {
		reader1 := strings.NewReader("content1")
		hash1, err1 := Sha256Reader(reader1)
		require.NoError(t, err1)

		reader2 := strings.NewReader("content2")
		hash2, err2 := Sha256Reader(reader2)
		require.NoError(t, err2)

		assert.NotEqual(t, hash1, hash2)
	})
}

func TestSha256File(t *testing.T) {
	t.Run("successful hash of file with content", func(t *testing.T) {
		tempFile := createTempFile(t, "hello world")
		defer os.Remove(tempFile)

		hash, err := Sha256File(tempFile)
		require.NoError(t, err)
		// Known SHA-256 hash of "hello world"
		assert.Equal(t, "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9", hash)
	})

	t.Run("successful hash of empty file", func(t *testing.T) {
		tempFile := createTempFile(t, "")
		defer os.Remove(tempFile)

		hash, err := Sha256File(tempFile)
		require.NoError(t, err)
		// Known SHA-256 hash of empty string
		assert.Equal(t, "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855", hash)
	})

	t.Run("error when file does not exist", func(t *testing.T) {
		hash, err := Sha256File("/nonexistent/path/to/file.txt")
		assert.Error(t, err)
		assert.Empty(t, hash)
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("error when path is a directory", func(t *testing.T) {
		tempDir := t.TempDir()

		hash, err := Sha256File(tempDir)
		assert.Error(t, err)
		assert.Empty(t, hash)
	})

	t.Run("successful hash of file with multiline content", func(t *testing.T) {
		content := "line1\nline2\nline3\n"
		tempFile := createTempFile(t, content)
		defer os.Remove(tempFile)

		hash, err := Sha256File(tempFile)
		require.NoError(t, err)
		assert.Len(t, hash, 64)
		assert.NotEmpty(t, hash)
	})

	t.Run("successful hash of large file", func(t *testing.T) {
		// Create a large file (1MB)
		largeContent := strings.Repeat("test", 256*1024)
		tempFile := createTempFile(t, largeContent)
		defer os.Remove(tempFile)

		hash, err := Sha256File(tempFile)
		require.NoError(t, err)
		assert.Len(t, hash, 64)
	})

	t.Run("same file content produces same hash", func(t *testing.T) {
		content := "consistent content"

		tempFile1 := createTempFile(t, content)
		defer os.Remove(tempFile1)

		tempFile2 := createTempFile(t, content)
		defer os.Remove(tempFile2)

		hash1, err1 := Sha256File(tempFile1)
		require.NoError(t, err1)

		hash2, err2 := Sha256File(tempFile2)
		require.NoError(t, err2)

		assert.Equal(t, hash1, hash2)
	})

	t.Run("different file content produces different hash", func(t *testing.T) {
		tempFile1 := createTempFile(t, "content1")
		defer os.Remove(tempFile1)

		tempFile2 := createTempFile(t, "content2")
		defer os.Remove(tempFile2)

		hash1, err1 := Sha256File(tempFile1)
		require.NoError(t, err1)

		hash2, err2 := Sha256File(tempFile2)
		require.NoError(t, err2)

		assert.NotEqual(t, hash1, hash2)
	})

	t.Run("successful hash of file with special characters", func(t *testing.T) {
		content := "Special: é, ñ, 中文, 🎉"
		tempFile := createTempFile(t, content)
		defer os.Remove(tempFile)

		hash, err := Sha256File(tempFile)
		require.NoError(t, err)
		assert.Len(t, hash, 64)
	})

	t.Run("successful hash of file with binary content", func(t *testing.T) {
		content := "\x00\x01\x02\x03\xff\xfe\xfd\xfc"
		tempFile := createTempFile(t, content)
		defer os.Remove(tempFile)

		hash, err := Sha256File(tempFile)
		require.NoError(t, err)
		assert.Len(t, hash, 64)
	})

	t.Run("error with invalid path characters", func(t *testing.T) {
		hash, err := Sha256File("\x00invalid")
		assert.Error(t, err)
		assert.Empty(t, hash)
	})
}

func TestSha256Reader_Sha256File_Consistency(t *testing.T) {
	t.Run("reader and file produce same hash for same content", func(t *testing.T) {
		content := "consistency test content"

		// Hash via reader
		reader := strings.NewReader(content)
		readerHash, err := Sha256Reader(reader)
		require.NoError(t, err)

		// Hash via file
		tempFile := createTempFile(t, content)
		defer os.Remove(tempFile)
		fileHash, err := Sha256File(tempFile)
		require.NoError(t, err)

		assert.Equal(t, readerHash, fileHash)
	})
}

// createTempFile creates a temporary file with the given content and returns its path
func createTempFile(t *testing.T, content string) string {
	t.Helper()

	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "testfile.txt")

	err := os.WriteFile(tempFile, []byte(content), 0644)
	require.NoError(t, err)

	return tempFile
}
