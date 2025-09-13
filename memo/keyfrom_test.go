package memo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestKeyFrom(t *testing.T) {
	t.Run("single string parameter", func(t *testing.T) {
		key := KeyFrom("test")
		assert.NotEmpty(t, key)
		assert.Len(t, key, 64) // SHA-256 hex string is 64 characters

		// Same input should produce same key
		key2 := KeyFrom("test")
		assert.Equal(t, key, key2)
	})

	t.Run("single int parameter", func(t *testing.T) {
		key := KeyFrom(42)
		assert.NotEmpty(t, key)
		assert.Len(t, key, 64)

		// Same input should produce same key
		key2 := KeyFrom(42)
		assert.Equal(t, key, key2)
	})

	t.Run("multiple parameters", func(t *testing.T) {
		key := KeyFrom("test", 42, true)
		assert.NotEmpty(t, key)
		assert.Len(t, key, 64)

		// Same inputs should produce same key
		key2 := KeyFrom("test", 42, true)
		assert.Equal(t, key, key2)
	})

	t.Run("parameter order matters", func(t *testing.T) {
		key1 := KeyFrom("a", "b")
		key2 := KeyFrom("b", "a")
		assert.NotEqual(t, key1, key2)
	})

	t.Run("different types with same values", func(t *testing.T) {
		key1 := KeyFrom(42)
		key2 := KeyFrom("42")
		assert.NotEqual(t, key1, key2)
	})

	t.Run("empty parameters", func(t *testing.T) {
		key := KeyFrom()
		assert.NotEmpty(t, key)
		assert.Len(t, key, 64)

		// Should be consistent
		key2 := KeyFrom()
		assert.Equal(t, key, key2)
	})

	t.Run("nil parameter", func(t *testing.T) {
		key := KeyFrom(nil)
		assert.NotEmpty(t, key)
		assert.Len(t, key, 64)

		// Should be consistent
		key2 := KeyFrom(nil)
		assert.Equal(t, key, key2)
	})

	t.Run("complex types", func(t *testing.T) {
		type testStruct struct {
			Name  string
			Value int
		}

		s := testStruct{Name: "test", Value: 42}
		key := KeyFrom(s)
		assert.NotEmpty(t, key)
		assert.Len(t, key, 64)

		// Same struct should produce same key
		key2 := KeyFrom(s)
		assert.Equal(t, key, key2)

		// Different struct should produce different key
		s2 := testStruct{Name: "test", Value: 43}
		key3 := KeyFrom(s2)
		assert.NotEqual(t, key, key3)
	})

	t.Run("slice parameters", func(t *testing.T) {
		slice1 := []int{1, 2, 3}
		slice2 := []int{1, 2, 3}
		slice3 := []int{3, 2, 1}

		key1 := KeyFrom(slice1)
		key2 := KeyFrom(slice2)
		key3 := KeyFrom(slice3)

		assert.Equal(t, key1, key2)
		assert.NotEqual(t, key1, key3)
	})

	t.Run("map parameters", func(t *testing.T) {
		map1 := map[string]int{"a": 1, "b": 2}
		map2 := map[string]int{"a": 1, "b": 2}
		map3 := map[string]int{"a": 2, "b": 1}

		key1 := KeyFrom(map1)
		key2 := KeyFrom(map2)
		key3 := KeyFrom(map3)

		assert.Equal(t, key1, key2)
		assert.NotEqual(t, key1, key3)
	})

	t.Run("mixed types", func(t *testing.T) {
		key := KeyFrom("string", 42, true, []int{1, 2}, map[string]int{"key": 1})
		assert.NotEmpty(t, key)
		assert.Len(t, key, 64)

		// Should be reproducible
		key2 := KeyFrom("string", 42, true, []int{1, 2}, map[string]int{"key": 1})
		assert.Equal(t, key, key2)
	})

	t.Run("returns valid hex string", func(t *testing.T) {
		key := KeyFrom("test")

		// Should be valid hex
		for _, c := range key {
			assert.True(t, (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f'),
				"Key should contain only hex characters, found: %c", c)
		}
	})

	t.Run("large number of parameters", func(t *testing.T) {
		var parts []any
		for i := 0; i < 100; i++ {
			parts = append(parts, i)
		}

		key := KeyFrom(parts...)
		assert.NotEmpty(t, key)
		assert.Len(t, key, 64)

		// Should be reproducible
		key2 := KeyFrom(parts...)
		assert.Equal(t, key, key2)
	})

	t.Run("consistency across calls", func(t *testing.T) {
		// Test that the function is deterministic across multiple calls
		inputs := []any{"test", 123, true, []string{"a", "b"}}

		var keys []string
		for i := 0; i < 10; i++ {
			key := KeyFrom(inputs...)
			keys = append(keys, key)
		}

		// All keys should be identical
		for i := 1; i < len(keys); i++ {
			assert.Equal(t, keys[0], keys[i])
		}
	})
}

func BenchmarkKeyFrom(b *testing.B) {
	b.Run("single string", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			KeyFrom("test")
		}
	})

	b.Run("multiple parameters", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			KeyFrom("test", 42, true, []int{1, 2, 3})
		}
	})

	b.Run("complex struct", func(b *testing.B) {
		type complexStruct struct {
			Name   string
			Values []int
			Meta   map[string]string
		}

		s := complexStruct{
			Name:   "benchmark",
			Values: []int{1, 2, 3, 4, 5},
			Meta:   map[string]string{"key1": "value1", "key2": "value2"},
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			KeyFrom(s)
		}
	})
}
