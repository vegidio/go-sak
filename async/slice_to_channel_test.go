package async

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSliceToChannel_IntToInt(t *testing.T) {
	// Given
	numbers := []int{1, 2, 3, 4, 5}
	doubleFunc := func(n int) int { return n * 2 }

	// When
	ch := SliceToChannel(numbers, 1, doubleFunc)

	// Then
	var results []int
	for result := range ch {
		results = append(results, result)
	}

	expected := []int{2, 4, 6, 8, 10}
	assert.Equal(t, expected, results)
}

func TestSliceToChannel_IntToString(t *testing.T) {
	// Given
	numbers := []int{1, 2, 3}
	toStringFunc := func(n int) string { return string(rune(n + 64)) }

	// When
	ch := SliceToChannel(numbers, 1, toStringFunc)

	// Then
	var results []string
	for result := range ch {
		results = append(results, result)
	}

	expected := []string{"A", "B", "C"}
	assert.Equal(t, expected, results)
}

func TestSliceToChannel_StringToInt(t *testing.T) {
	// Given
	strings := []string{"hello", "world", "test"}
	lengthFunc := func(s string) int { return len(s) }

	// When
	ch := SliceToChannel(strings, 1, lengthFunc)

	// Then
	var results []int
	for result := range ch {
		results = append(results, result)
	}

	expected := []int{5, 5, 4}
	assert.Equal(t, expected, results)
}

func TestSliceToChannel_EmptySlice(t *testing.T) {
	// Given
	empty := []int{}
	identityFunc := func(n int) int { return n }

	// When
	ch := SliceToChannel(empty, 1, identityFunc)

	// Then
	var results []int
	for result := range ch {
		results = append(results, result)
	}

	assert.Empty(t, results)
}

func TestSliceToChannel_SingleElement(t *testing.T) {
	// Given
	single := []int{42}
	squareFunc := func(n int) int { return n * n }

	// When
	ch := SliceToChannel(single, 1, squareFunc)

	// Then
	var results []int
	for result := range ch {
		results = append(results, result)
	}

	expected := []int{1764}
	assert.Equal(t, expected, results)
}

func TestSliceToChannel_PreservesOrder(t *testing.T) {
	// Given
	numbers := []int{5, 3, 8, 1, 9, 2}
	identityFunc := func(n int) int { return n }

	// When
	ch := SliceToChannel(numbers, 1, identityFunc)

	// Then
	var results []int
	for result := range ch {
		results = append(results, result)
	}

	assert.Equal(t, numbers, results)
}

func TestSliceToChannel_ChannelIsClosed(t *testing.T) {
	// Given
	numbers := []int{1, 2, 3}
	identityFunc := func(n int) int { return n }

	// When
	ch := SliceToChannel(numbers, 1, identityFunc)

	// Consume all elements
	for range ch {
	}

	// Then - verify if the channel is closed
	_, ok := <-ch
	assert.False(t, ok, "Channel should be closed after all items are processed")
}

func TestSliceToChannel_ComplexTransformation(t *testing.T) {
	// Given
	type Person struct {
		Name string
		Age  int
	}
	people := []Person{
		{Name: "Alice", Age: 30},
		{Name: "Bob", Age: 25},
		{Name: "Charlie", Age: 35},
	}
	nameFunc := func(p Person) string { return p.Name }

	// When
	ch := SliceToChannel(people, 1, nameFunc)

	// Then
	var results []string
	for result := range ch {
		results = append(results, result)
	}

	expected := []string{"Alice", "Bob", "Charlie"}
	assert.Equal(t, expected, results)
}

func TestSliceToChannel_ReturnsReadOnlyChannel(t *testing.T) {
	// Given
	numbers := []int{1, 2, 3}
	identityFunc := func(n int) int { return n }

	// When
	ch := SliceToChannel(numbers, 1, identityFunc)

	// Then - verify it's a read-only channel (compile-time check)
	// This is implicitly tested by the return type <-chan R
	// We can verify we can read from it
	result, ok := <-ch
	assert.True(t, ok)
	assert.Equal(t, 1, result)
}
