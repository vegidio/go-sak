package crypto

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestXxh3Bytes(t *testing.T) {
	t.Run("empty byte slice", func(t *testing.T) {
		hash, err := Xxh3Bytes([]byte{})
		require.NoError(t, err)
		// XXH3 of empty input
		assert.Equal(t, "99aa06d3014798d86001c324468d497f", hash)
	})

	t.Run("simple text", func(t *testing.T) {
		hash, err := Xxh3Bytes([]byte("hello world"))
		require.NoError(t, err)
		assert.Equal(t, "df8d09e93f874900a99b8775cc15b6c7", hash)
	})

	t.Run("single character", func(t *testing.T) {
		hash, err := Xxh3Bytes([]byte("a"))
		require.NoError(t, err)
		assert.Equal(t, "a96faf705af16834e6c632b61e964e1f", hash)
	})

	t.Run("binary data", func(t *testing.T) {
		hash, err := Xxh3Bytes([]byte{0x00, 0x01, 0x02, 0x03, 0xFF})
		require.NoError(t, err)
		assert.Len(t, hash, 32) // XXH3-128 produces 32 hex characters
		assert.NotEmpty(t, hash)
	})

	t.Run("text with special characters", func(t *testing.T) {
		hash, err := Xxh3Bytes([]byte("Hello, 世界! 🌍"))
		require.NoError(t, err)
		// Verify hash is valid hexadecimal and correct length
		assert.Len(t, hash, 32) // XXH3-128 produces 32 hex characters
		assert.NotEmpty(t, hash)
	})

	t.Run("long input", func(t *testing.T) {
		longBytes := []byte(strings.Repeat("a", 10000))
		hash, err := Xxh3Bytes(longBytes)
		require.NoError(t, err)
		assert.Len(t, hash, 32)
		assert.NotEmpty(t, hash)
	})

	t.Run("newline characters", func(t *testing.T) {
		hash, err := Xxh3Bytes([]byte("line1\nline2\r\nline3"))
		require.NoError(t, err)
		assert.Len(t, hash, 32)
		assert.NotEmpty(t, hash)
	})

	t.Run("nil byte slice", func(t *testing.T) {
		hash, err := Xxh3Bytes(nil)
		require.NoError(t, err)
		// nil slice is treated same as empty slice
		assert.Equal(t, "99aa06d3014798d86001c324468d497f", hash)
	})

	t.Run("numeric bytes", func(t *testing.T) {
		hash, err := Xxh3Bytes([]byte("123456"))
		require.NoError(t, err)
		assert.Len(t, hash, 32)
		assert.NotEmpty(t, hash)
	})
}

func TestXxh3String(t *testing.T) {
	t.Run("empty string", func(t *testing.T) {
		hash, err := Xxh3String("")
		require.NoError(t, err)
		assert.Equal(t, "99aa06d3014798d86001c324468d497f", hash)
	})

	t.Run("hello world example from docs", func(t *testing.T) {
		hash, err := Xxh3String("hello world")
		require.NoError(t, err)
		assert.Equal(t, "df8d09e93f874900a99b8775cc15b6c7", hash)
	})

	t.Run("single character string", func(t *testing.T) {
		hash, err := Xxh3String("a")
		require.NoError(t, err)
		assert.Equal(t, "a96faf705af16834e6c632b61e964e1f", hash)
	})

	t.Run("string with spaces", func(t *testing.T) {
		hash, err := Xxh3String("hello world with spaces")
		require.NoError(t, err)
		assert.Len(t, hash, 32)
		assert.NotEmpty(t, hash)
	})

	t.Run("unicode string", func(t *testing.T) {
		hash, err := Xxh3String("你好世界")
		require.NoError(t, err)
		assert.Len(t, hash, 32)
		assert.NotEmpty(t, hash)
	})

	t.Run("string with emoji", func(t *testing.T) {
		hash, err := Xxh3String("Hello 👋 World 🌍")
		require.NoError(t, err)
		assert.Len(t, hash, 32)
		assert.NotEmpty(t, hash)
	})

	t.Run("very long string", func(t *testing.T) {
		longString := strings.Repeat("test", 10000)
		hash, err := Xxh3String(longString)
		require.NoError(t, err)
		assert.Len(t, hash, 32)
		assert.NotEmpty(t, hash)
	})

	t.Run("string with special characters", func(t *testing.T) {
		hash, err := Xxh3String("!@#$%^&*()_+-=[]{}|;':\",./<>?")
		require.NoError(t, err)
		assert.Len(t, hash, 32)
		assert.NotEmpty(t, hash)
	})

	t.Run("numeric string", func(t *testing.T) {
		hash, err := Xxh3String("1234567890")
		require.NoError(t, err)
		assert.Len(t, hash, 32)
		assert.NotEmpty(t, hash)
	})

	t.Run("consistency between Xxh3String and Xxh3Bytes", func(t *testing.T) {
		testInput := "consistency test"

		hashFromString, err1 := Xxh3String(testInput)
		require.NoError(t, err1)

		hashFromBytes, err2 := Xxh3Bytes([]byte(testInput))
		require.NoError(t, err2)

		assert.Equal(t, hashFromString, hashFromBytes)
	})
}

func TestXxh3EdgeCases(t *testing.T) {
	t.Run("hash output is always 32 characters", func(t *testing.T) {
		testCases := []string{"", "a", "short", strings.Repeat("long", 1000)}

		for _, tc := range testCases {
			hash, err := Xxh3String(tc)
			require.NoError(t, err)
			assert.Len(t, hash, 32, "hash length should always be 32 for input: %q", tc)
		}
	})

	t.Run("hash output is always lowercase hexadecimal", func(t *testing.T) {
		hash, err := Xxh3String("test")
		require.NoError(t, err)

		for _, char := range hash {
			assert.True(t,
				(char >= '0' && char <= '9') || (char >= 'a' && char <= 'f'),
				"hash should only contain lowercase hex characters, found: %c", char)
		}
	})

	t.Run("different inputs produce different hashes", func(t *testing.T) {
		hash1, err1 := Xxh3String("test1")
		require.NoError(t, err1)

		hash2, err2 := Xxh3String("test2")
		require.NoError(t, err2)

		assert.NotEqual(t, hash1, hash2)
	})

	t.Run("same input produces same hash", func(t *testing.T) {
		input := "deterministic test"

		hash1, err1 := Xxh3String(input)
		require.NoError(t, err1)

		hash2, err2 := Xxh3String(input)
		require.NoError(t, err2)

		assert.Equal(t, hash1, hash2)
	})
}
