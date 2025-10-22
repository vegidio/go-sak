package async

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMapChannel_IntToInt(t *testing.T) {
	// Given
	numbers := []int{1, 2, 3, 4, 5}
	doubleFunc := func(n int) int { return n * 2 }

	// When
	ch := MapChannel(numbers, doubleFunc)

	// Then
	var results []int
	for result := range ch {
		results = append(results, result)
	}

	expected := []int{2, 4, 6, 8, 10}
	assert.Equal(t, expected, results)
}

func TestMapChannel_IntToString(t *testing.T) {
	// Given
	numbers := []int{1, 2, 3}
	toStringFunc := func(n int) string { return string(rune(n + 64)) }

	// When
	ch := MapChannel(numbers, toStringFunc)

	// Then
	var results []string
	for result := range ch {
		results = append(results, result)
	}

	expected := []string{"A", "B", "C"}
	assert.Equal(t, expected, results)
}

func TestMapChannel_StringToInt(t *testing.T) {
	// Given
	strings := []string{"hello", "world", "test"}
	lengthFunc := func(s string) int { return len(s) }

	// When
	ch := MapChannel(strings, lengthFunc)

	// Then
	var results []int
	for result := range ch {
		results = append(results, result)
	}

	expected := []int{5, 5, 4}
	assert.Equal(t, expected, results)
}

func TestMapChannel_EmptySlice(t *testing.T) {
	// Given
	empty := []int{}
	identityFunc := func(n int) int { return n }

	// When
	ch := MapChannel(empty, identityFunc)

	// Then
	var results []int
	for result := range ch {
		results = append(results, result)
	}

	assert.Empty(t, results)
}

func TestMapChannel_SingleElement(t *testing.T) {
	// Given
	single := []int{42}
	squareFunc := func(n int) int { return n * n }

	// When
	ch := MapChannel(single, squareFunc)

	// Then
	var results []int
	for result := range ch {
		results = append(results, result)
	}

	expected := []int{1764}
	assert.Equal(t, expected, results)
}

func TestMapChannel_PreservesOrder(t *testing.T) {
	// Given
	numbers := []int{5, 3, 8, 1, 9, 2}
	identityFunc := func(n int) int { return n }

	// When
	ch := MapChannel(numbers, identityFunc)

	// Then
	var results []int
	for result := range ch {
		results = append(results, result)
	}

	assert.Equal(t, numbers, results)
}

func TestMapChannel_ChannelIsClosed(t *testing.T) {
	// Given
	numbers := []int{1, 2, 3}
	identityFunc := func(n int) int { return n }

	// When
	ch := MapChannel(numbers, identityFunc)

	// Consume all elements
	for range ch {
	}

	// Then - verify if the channel is closed
	_, ok := <-ch
	assert.False(t, ok, "Channel should be closed after all items are processed")
}

func TestMapChannel_ComplexTransformation(t *testing.T) {
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
	ch := MapChannel(people, nameFunc)

	// Then
	var results []string
	for result := range ch {
		results = append(results, result)
	}

	expected := []string{"Alice", "Bob", "Charlie"}
	assert.Equal(t, expected, results)
}

func TestMapChannel_ReturnsReadOnlyChannel(t *testing.T) {
	// Given
	numbers := []int{1, 2, 3}
	identityFunc := func(n int) int { return n }

	// When
	ch := MapChannel(numbers, identityFunc)

	// Then - verify it's a read-only channel (compile-time check)
	// This is implicitly tested by the return type <-chan R
	// We can verify we can read from it
	result, ok := <-ch
	assert.True(t, ok)
	assert.Equal(t, 1, result)
}
