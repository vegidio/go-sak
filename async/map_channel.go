package async

// MapChannel applies a transformation function to each element in a slice and returns a read-only channel that emits
// the transformed results.
//
// The function processes items asynchronously in a separate goroutine. Each item from the input slice is transformed
// using the provided function and sent to the output channel in order. The channel is automatically closed when all
// items have been processed.
//
// # Type Parameters:
//   - T: the type of elements in the input slice
//   - R: the type of elements returned by the transformation function and emitted
//     on the channel
//
// # Parameters:
//   - items: the slice of items to transform
//   - fn: the transformation function to apply to each item
//
// # Returns:
//   - A read-only channel that emits transformed results
//
// # Example:
//
//	numbers := []int{1, 2, 3, 4, 5}
//	ch := MapChannel(numbers, func(n int) int { return n * 2 })
//	for result := range ch {
//	    fmt.Println(result) // outputs: 2, 4, 6, 8, 10
//	}
func MapChannel[T any, R any](items []T, fn func(T) R) <-chan R {
	out := make(chan R)

	go func() {
		defer close(out)
		for _, v := range items {
			out <- fn(v)
		}
	}()

	return out
}
