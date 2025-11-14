package async

import "sync"

// ConcurrentChannel processes items from an input channel concurrently and returns a channel of results. It spawns the
// specified number of worker goroutines to apply the given function to each item.
//
// # Type parameters:
//   - T: the type of items in the input channel
//   - R: the type of items in the output channel (result type)
//
// # Parameters:
//   - input: a receive-only channel from which items of type T are read
//   - concurrency: the number of worker goroutines to spawn for parallel processing
//   - fn: a function that transforms an item of type T into a result of type R
//
// # Returns:
//   - a receive-only channel that emits results of type R. The channel is automatically
//     closed when all items from the input channel have been processed.
//
// # Example:
//
//	numbers := make(chan int)
//	go func() {
//		for i := 1; i <= 10; i++ {
//			numbers <- i
//		}
//		close(numbers)
//	}()
//
//	squares := ConcurrentChannel(numbers, 3, func(n int) int {
//		return n * n
//	})
//
//	for result := range squares {
//		fmt.Println(result)
//	}
//
// Note: The order of results in the output channel is not guaranteed to match
// the order of items in the input channel due to concurrent processing.
func ConcurrentChannel[T any, R any](input <-chan T, concurrency int, fn func(T) R) <-chan R {
	output := make(chan R)

	var wg sync.WaitGroup
	wg.Add(concurrency)

	for i := 0; i < concurrency; i++ {
		go func() {
			defer wg.Done()
			for item := range input {
				output <- fn(item)
			}
		}()
	}

	// Close output once all workers are done
	go func() {
		wg.Wait()
		close(output)
	}()

	return output
}
