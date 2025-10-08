package async

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProcessChannel_BasicFunctionality(t *testing.T) {
	// Given
	input := make(chan int)
	expected := map[int]bool{1: true, 4: true, 9: true, 16: true, 25: true}

	go func() {
		for i := 1; i <= 5; i++ {
			input <- i
		}
		close(input)
	}()

	// When
	output := ProcessChannel(input, 2, func(n int) int {
		return n * n
	})

	// Then
	results := make(map[int]bool)
	for result := range output {
		results[result] = true
	}

	assert.Equal(t, expected, results)
}

func TestProcessChannel_EmptyInput(t *testing.T) {
	// Given
	input := make(chan int)
	close(input)

	// When
	output := ProcessChannel(input, 3, func(n int) int {
		return n * 2
	})

	// Then
	var results []int
	for result := range output {
		results = append(results, result)
	}

	assert.Empty(t, results)
}

func TestProcessChannel_SingleWorker(t *testing.T) {
	// Given
	input := make(chan string)
	expected := map[string]bool{"hello": true, "world": true, "test": true}

	go func() {
		input <- "hello"
		input <- "world"
		input <- "test"
		close(input)
	}()

	// When
	output := ProcessChannel(input, 1, func(s string) string {
		return s
	})

	// Then
	results := make(map[string]bool)
	for result := range output {
		results[result] = true
	}

	assert.Equal(t, expected, results)
}

func TestProcessChannel_MultipleWorkers(t *testing.T) {
	// Given
	input := make(chan int)
	itemCount := 100
	concurrency := 10

	go func() {
		for i := 0; i < itemCount; i++ {
			input <- i
		}
		close(input)
	}()

	// When
	output := ProcessChannel(input, concurrency, func(n int) int {
		return n * 2
	})

	// Then
	results := make(map[int]bool)
	for result := range output {
		results[result] = true
	}

	assert.Len(t, results, itemCount)
	for i := 0; i < itemCount; i++ {
		assert.True(t, results[i*2], "Expected %d to be in results", i*2)
	}
}

func TestProcessChannel_DifferentTypes(t *testing.T) {
	// Given
	input := make(chan int)

	go func() {
		for i := 1; i <= 3; i++ {
			input <- i
		}
		close(input)
	}()

	// When
	output := ProcessChannel(input, 2, func(n int) string {
		return string(rune('A' + n - 1))
	})

	// Then
	results := make(map[string]bool)
	for result := range output {
		results[result] = true
	}

	expected := map[string]bool{"A": true, "B": true, "C": true}
	assert.Equal(t, expected, results)
}

func TestProcessChannel_OutputChannelCloses(t *testing.T) {
	// Given
	input := make(chan int)

	go func() {
		for i := 0; i < 5; i++ {
			input <- i
		}
		close(input)
	}()

	// When
	output := ProcessChannel(input, 2, func(n int) int {
		return n
	})

	// Then
	count := 0
	for range output {
		count++
	}

	assert.Equal(t, 5, count)

	// Verify if the channel is closed
	_, ok := <-output
	assert.False(t, ok, "Output channel should be closed")
}

func TestProcessChannel_ConcurrentProcessing(t *testing.T) {
	// Given
	input := make(chan int)
	processed := make(map[int]bool)
	var mu sync.Mutex
	concurrency := 5

	go func() {
		for i := 0; i < 50; i++ {
			input <- i
		}
		close(input)
	}()

	// When
	output := ProcessChannel(input, concurrency, func(n int) int {
		mu.Lock()
		processed[n] = true
		mu.Unlock()
		time.Sleep(time.Millisecond) // Simulate some work
		return n * 2
	})

	// Then
	var results []int
	for result := range output {
		results = append(results, result)
	}

	assert.Len(t, results, 50)
	assert.Len(t, processed, 50)
}

func TestProcessChannel_SlowProducer(t *testing.T) {
	// Given
	input := make(chan int)

	go func() {
		for i := 0; i < 5; i++ {
			time.Sleep(10 * time.Millisecond)
			input <- i
		}
		close(input)
	}()

	// When
	start := time.Now()
	output := ProcessChannel(input, 3, func(n int) int {
		return n
	})

	// Then
	count := 0
	for range output {
		count++
	}

	duration := time.Since(start)
	assert.Equal(t, 5, count)
	assert.GreaterOrEqual(t, duration, 50*time.Millisecond)
}

func TestProcessChannel_SlowFunction(t *testing.T) {
	// Given
	input := make(chan int)

	go func() {
		for i := 0; i < 10; i++ {
			input <- i
		}
		close(input)
	}()

	// When
	start := time.Now()
	output := ProcessChannel(input, 5, func(n int) int {
		time.Sleep(10 * time.Millisecond)
		return n
	})

	// Then
	count := 0
	for range output {
		count++
	}

	duration := time.Since(start)
	assert.Equal(t, 10, count)
	// With 5 workers and 10 items taking 10ms each, should take ~20ms
	assert.Less(t, duration, 50*time.Millisecond)
}

func TestProcessChannel_ZeroConcurrency(t *testing.T) {
	// Given
	input := make(chan int)

	go func() {
		input <- 1
		input <- 2
		close(input)
	}()

	// When
	output := ProcessChannel(input, 0, func(n int) int {
		return n * 2
	})

	// Then
	var results []int
	done := make(chan bool)

	go func() {
		for result := range output {
			results = append(results, result)
		}
		done <- true
	}()

	select {
	case <-done:
		assert.Empty(t, results, "Should process nothing with 0 workers")
	case <-time.After(100 * time.Millisecond):
		assert.Empty(t, results, "Should process nothing with 0 workers")
	}
}

func TestProcessChannel_ComplexStructs(t *testing.T) {
	// Given
	type Person struct {
		Name string
		Age  int
	}
	type Result struct {
		FullName string
		IsAdult  bool
	}

	input := make(chan Person)

	go func() {
		input <- Person{Name: "Alice", Age: 25}
		input <- Person{Name: "Bob", Age: 17}
		input <- Person{Name: "Charlie", Age: 30}
		close(input)
	}()

	// When
	output := ProcessChannel(input, 2, func(p Person) Result {
		return Result{
			FullName: "Mr/Ms " + p.Name,
			IsAdult:  p.Age >= 18,
		}
	})

	// Then
	var results []Result
	for result := range output {
		results = append(results, result)
	}

	require.Len(t, results, 3)

	adults := 0
	for _, r := range results {
		assert.Contains(t, r.FullName, "Mr/Ms")
		if r.IsAdult {
			adults++
		}
	}
	assert.Equal(t, 2, adults)
}
