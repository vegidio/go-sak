package crypto

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSha256File(t *testing.T) {
	// Create a temporary directory for test files
	tempDir := t.TempDir()

	t.Run("successful hash calculation for non-empty file", func(t *testing.T) {
		// Create a test file with known content
		testContent := "Hello, World!"
		testFile := filepath.Join(tempDir, "test.txt")

		err := os.WriteFile(testFile, []byte(testContent), 0644)
		require.NoError(t, err)

		// Calculate hash
		hash, err := Sha256File(testFile)

		// Assert no error and correct hash
		assert.NoError(t, err)
		// SHA-256 of "Hello, World!" is dffd6021bb2bd5b0af676290809ec3a53191dd81c7f70a4b28688a362182986f
		expected := "dffd6021bb2bd5b0af676290809ec3a53191dd81c7f70a4b28688a362182986f"
		assert.Equal(t, expected, hash)
		assert.Len(t, hash, 64) // SHA-256 hash should be 64 hex characters
	})

	t.Run("successful hash calculation for empty file", func(t *testing.T) {
		// Create an empty test file
		testFile := filepath.Join(tempDir, "empty.txt")

		err := os.WriteFile(testFile, []byte(""), 0644)
		require.NoError(t, err)

		// Calculate hash
		hash, err := Sha256File(testFile)

		// Assert no error and correct hash for an empty file
		assert.NoError(t, err)
		// SHA-256 of the empty string is e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
		expected := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
		assert.Equal(t, expected, hash)
		assert.Len(t, hash, 64)
	})

	t.Run("successful hash calculation for large file", func(t *testing.T) {
		// Create a larger test file
		testContent := make([]byte, 10000)
		for i := range testContent {
			testContent[i] = byte(i % 256)
		}
		testFile := filepath.Join(tempDir, "large.bin")

		err := os.WriteFile(testFile, testContent, 0644)
		require.NoError(t, err)

		// Calculate hash
		hash, err := Sha256File(testFile)

		// Assert no error and valid hash format
		assert.NoError(t, err)
		assert.Len(t, hash, 64)
		assert.Regexp(t, "^[a-f0-9]+$", hash) // Should be lowercase hex
	})

	t.Run("file does not exist", func(t *testing.T) {
		nonExistentFile := filepath.Join(tempDir, "does_not_exist.txt")

		hash, err := Sha256File(nonExistentFile)

		// Assert error occurred and empty hash returned
		assert.Error(t, err)
		assert.Empty(t, hash)
		assert.Contains(t, err.Error(), "no such file or directory")
	})

	t.Run("file path is empty", func(t *testing.T) {
		hash, err := Sha256File("")

		// Assert error occurred and empty hash returned
		assert.Error(t, err)
		assert.Empty(t, hash)
	})

	t.Run("file path is directory", func(t *testing.T) {
		hash, err := Sha256File(tempDir)

		// Assert error occurred and empty hash returned
		assert.Error(t, err)
		assert.Empty(t, hash)
	})

	t.Run("hash format is lowercase hexadecimal", func(t *testing.T) {
		// Create a test file
		testContent := "test content for format validation"
		testFile := filepath.Join(tempDir, "format_test.txt")

		err := os.WriteFile(testFile, []byte(testContent), 0644)
		require.NoError(t, err)

		// Calculate hash
		hash, err := Sha256File(testFile)

		// Assert no error and correct format
		assert.NoError(t, err)
		assert.Regexp(t, "^[a-f0-9]{64}$", hash) // Should be exactly 64 lowercase hex characters
	})

	t.Run("consistent results for same file", func(t *testing.T) {
		// Create a test file
		testContent := "consistency test"
		testFile := filepath.Join(tempDir, "consistency.txt")

		err := os.WriteFile(testFile, []byte(testContent), 0644)
		require.NoError(t, err)

		// Calculate hash multiple times
		hash1, err1 := Sha256File(testFile)
		hash2, err2 := Sha256File(testFile)
		hash3, err3 := Sha256File(testFile)

		// Assert all calls succeed and return the same result
		assert.NoError(t, err1)
		assert.NoError(t, err2)
		assert.NoError(t, err3)
		assert.Equal(t, hash1, hash2)
		assert.Equal(t, hash2, hash3)
	})
}
