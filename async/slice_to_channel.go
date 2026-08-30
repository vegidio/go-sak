package async

import "sync"

// SliceToChannel applies a transformation function to each element in a slice concurrently and returns a read-only
// channel that emits the transformed results.
//
// The function processes items in parallel using the specified level of concurrency. Each item from the input slice is
// transformed using the provided function and sent to the output channel. The channel is automatically closed when all
// items have been processed.
//
// # Type Parameters:
//   - T: the type of elements in the input slice
//   - R: the type of elements returned by the transformation function and emitted
//     on the channel
//
// # Parameters:
//   - items: the slice of items to transform
//   - concurrency: the maximum number of items to process simultaneously
//   - fn: the transformation function to apply to each item
//
// # Returns:
//   - A read-only channel that emits transformed results
//
// # Example:
//
//	numbers := []int{1, 2, 3, 4, 5}
//	ch := MapChannel(numbers, 2, func(n int) int { return n * 2 })
//	for result := range ch {
//	    fmt.Println(result) // outputs: 2, 4, 6, 8, 10 (order not guaranteed)
//	}
//
// Note: The order of results in the output channel is not guaranteed to match the order of items in the input slice due
// to concurrent processing.
func SliceToChannel[T any, R any](items []T, concurrency int, fn func(T) R) <-chan R {
	out := make(chan R)

	// A concurrency below 1 would make sem unbuffered, and the goroutine that drains it is only started after the
	// send succeeds - so the very first send would block forever, out would never close, and every consumer would
	// hang. A negative value would panic in make outright.
	if concurrency < 1 {
		concurrency = 1
	}

	go func() {
		defer close(out)
		var wg sync.WaitGroup
		sem := make(chan struct{}, concurrency)

		for _, v := range items {
			sem <- struct{}{}

			wg.Go(func() {
				defer func() { <-sem }()

				out <- fn(v)
			})
		}

		wg.Wait()
	}()

	return out
}
