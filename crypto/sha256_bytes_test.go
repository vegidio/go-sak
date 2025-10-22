package crypto

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSha256Bytes(t *testing.T) {
	t.Run("success with simple string bytes", func(t *testing.T) {
		// Act
		hash, err := Sha256Bytes([]byte("hello world"))

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9", hash)
	})

	t.Run("success with empty bytes", func(t *testing.T) {
		// Act
		hash, err := Sha256Bytes([]byte(""))

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855", hash)
	})

	t.Run("success with special characters", func(t *testing.T) {
		// Act
		hash, err := Sha256Bytes([]byte("Hello, 世界! 🌍"))

		// Assert
		assert.NoError(t, err)
		assert.NotEmpty(t, hash)
		assert.Len(t, hash, 64) // SHA-256 produces 64 hex characters
	})

	t.Run("success with newlines and tabs", func(t *testing.T) {
		// Act
		hash, err := Sha256Bytes([]byte("line1\nline2\tline3"))

		// Assert
		assert.NoError(t, err)
		assert.NotEmpty(t, hash)
		assert.Len(t, hash, 64)
	})

	t.Run("success with large byte slice", func(t *testing.T) {
		// Arrange
		longString := ""
		for i := 0; i < 10000; i++ {
			longString += "a"
		}

		// Act
		hash, err := Sha256Bytes([]byte(longString))

		// Assert
		assert.NoError(t, err)
		assert.NotEmpty(t, hash)
		assert.Len(t, hash, 64)
	})

	t.Run("consistency check - same input produces same hash", func(t *testing.T) {
		// Arrange
		input := []byte("test string for consistency")

		// Act
		hash1, err1 := Sha256Bytes(input)
		hash2, err2 := Sha256Bytes(input)

		// Assert
		assert.NoError(t, err1)
		assert.NoError(t, err2)
		assert.Equal(t, hash1, hash2)
	})

	t.Run("different inputs produce different hashes", func(t *testing.T) {
		// Act
		hash1, err1 := Sha256Bytes([]byte("string1"))
		hash2, err2 := Sha256Bytes([]byte("string2"))

		// Assert
		assert.NoError(t, err1)
		assert.NoError(t, err2)
		assert.NotEqual(t, hash1, hash2)
	})

	t.Run("hash format validation", func(t *testing.T) {
		// Act
		hash, err := Sha256Bytes([]byte("test"))

		// Assert
		assert.NoError(t, err)
		assert.Len(t, hash, 64, "SHA-256 hash should be 64 hex characters")
		assert.Regexp(t, "^[0-9a-f]{64}$", hash, "Hash should only contain lowercase hex characters")
	})

	t.Run("known test vectors", func(t *testing.T) {
		testCases := []struct {
			input    string
			expected string
		}{
			{
				input:    "abc",
				expected: "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad",
			},
			{
				input:    "The quick brown fox jumps over the lazy dog",
				expected: "d7a8fbb307d7809469ca9abcb0082e4f8d5651e46d3cdb762d02d0bf37c9e592",
			},
			{
				input:    "hello world",
				expected: "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.input, func(t *testing.T) {
				// Act
				hash, err := Sha256Bytes([]byte(tc.input))

				// Assert
				assert.NoError(t, err)
				assert.Equal(t, tc.expected, hash)
			})
		}
	})

	t.Run("binary data", func(t *testing.T) {
		// Arrange
		binaryString := "\x00\x01\x02\x03\xff\xfe"

		// Act
		hash, err := Sha256Bytes([]byte(binaryString))

		// Assert
		assert.NoError(t, err)
		assert.NotEmpty(t, hash)
		assert.Len(t, hash, 64)
	})

	t.Run("case sensitivity", func(t *testing.T) {
		// Act
		hashLower, err1 := Sha256Bytes([]byte("hello"))
		hashUpper, err2 := Sha256Bytes([]byte("HELLO"))

		// Assert
		assert.NoError(t, err1)
		assert.NoError(t, err2)
		assert.NotEqual(t, hashLower, hashUpper, "Hashes should differ for different cases")
	})

	t.Run("whitespace differences", func(t *testing.T) {
		// Act
		hash1, err1 := Sha256Bytes([]byte("hello world"))
		hash2, err2 := Sha256Bytes([]byte("hello  world")) // two spaces
		hash3, err3 := Sha256Bytes([]byte("helloworld"))

		// Assert
		assert.NoError(t, err1)
		assert.NoError(t, err2)
		assert.NoError(t, err3)
		assert.NotEqual(t, hash1, hash2)
		assert.NotEqual(t, hash1, hash3)
		assert.NotEqual(t, hash2, hash3)
	})

	t.Run("numeric strings", func(t *testing.T) {
		// Act
		hash1, err1 := Sha256Bytes([]byte("12345"))
		hash2, err2 := Sha256Bytes([]byte("123456"))

		// Assert
		assert.NoError(t, err1)
		assert.NoError(t, err2)
		assert.NotEqual(t, hash1, hash2)
	})
}
