package string

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRightOf(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		sub      string
		useLast  bool
		expected string
	}{
		{
			name:     "first occurrence with single delimiter",
			s:        "hello-world-test",
			sub:      "-",
			useLast:  false,
			expected: "world-test",
		},
		{
			name:     "last occurrence with single delimiter",
			s:        "hello-world-test",
			sub:      "-",
			useLast:  true,
			expected: "test",
		},
		{
			name:     "substring not found",
			s:        "hello",
			sub:      "x",
			useLast:  false,
			expected: "",
		},
		{
			name:     "substring not found with useLast true",
			s:        "hello",
			sub:      "x",
			useLast:  true,
			expected: "",
		},
		{
			name:     "empty string",
			s:        "",
			sub:      "-",
			useLast:  false,
			expected: "",
		},
		{
			name:     "empty substring",
			s:        "hello",
			sub:      "",
			useLast:  false,
			expected: "hello",
		},
		{
			name:     "substring at the beginning",
			s:        "-hello-world",
			sub:      "-",
			useLast:  false,
			expected: "hello-world",
		},
		{
			name:     "substring at the end",
			s:        "hello-world-",
			sub:      "-",
			useLast:  true,
			expected: "",
		},
		{
			name:     "multi-character substring first occurrence",
			s:        "hello::world::test",
			sub:      "::",
			useLast:  false,
			expected: "world::test",
		},
		{
			name:     "multi-character substring last occurrence",
			s:        "hello::world::test",
			sub:      "::",
			useLast:  true,
			expected: "test",
		},
		{
			name:     "substring is entire string",
			s:        "hello",
			sub:      "hello",
			useLast:  false,
			expected: "",
		},
		{
			name:     "substring longer than string",
			s:        "hi",
			sub:      "hello",
			useLast:  false,
			expected: "",
		},
		{
			name:     "single occurrence same behavior for useLast",
			s:        "hello-world",
			sub:      "-",
			useLast:  false,
			expected: "world",
		},
		{
			name:     "single occurrence same behavior for useLast true",
			s:        "hello-world",
			sub:      "-",
			useLast:  true,
			expected: "world",
		},
		{
			name:     "multiple occurrences of multi-char substring",
			s:        "one=>two=>three=>four",
			sub:      "=>",
			useLast:  false,
			expected: "two=>three=>four",
		},
		{
			name:     "multiple occurrences of multi-char substring last",
			s:        "one=>two=>three=>four",
			sub:      "=>",
			useLast:  true,
			expected: "four",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RightOf(tt.s, tt.sub, tt.useLast)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRightOf_EdgeCases(t *testing.T) {
	t.Run("both string and substring empty", func(t *testing.T) {
		result := RightOf("", "", false)
		assert.Equal(t, "", result)
	})

	t.Run("substring appears consecutively", func(t *testing.T) {
		result := RightOf("hello--world", "--", false)
		assert.Equal(t, "world", result)
	})

	t.Run("overlapping pattern first", func(t *testing.T) {
		result := RightOf("aaaa", "aa", false)
		assert.Equal(t, "aa", result)
	})

	t.Run("overlapping pattern last", func(t *testing.T) {
		result := RightOf("aaaa", "aa", true)
		assert.Equal(t, "", result)
	})
}
