package async

import (
	"context"
	"sync"
)

// ConcurrentChannelContext behaves like ConcurrentChannel, but stops as soon as the context is
// cancelled. Prefer it whenever the consumer may walk away before the input channel is drained:
// ConcurrentChannel blocks forever on its output send in that situation, leaking every worker.
//
// # Type parameters:
//   - T: the type of items in the input channel
//   - R: the type of items in the output channel (result type)
//
// # Parameters:
//   - ctx: context used to abort the processing
//   - input: a receive-only channel from which items of type T are read
//   - concurrency: the number of worker goroutines to spawn for parallel processing
//   - fn: a function that transforms an item of type T into a result of type R
//
// # Returns:
//   - a receive-only channel that emits results of type R. The channel is closed once all items have
//     been processed or the context is cancelled.
//
// Note: The order of results in the output channel is not guaranteed to match the order of items in
// the input channel due to concurrent processing.
func ConcurrentChannelContext[T any, R any](
	ctx context.Context,
	input <-chan T,
	concurrency int,
	fn func(T) R,
) <-chan R {
	output := make(chan R)

	// With a concurrency below 1 no worker would ever start: output would close immediately and the whole input
	// would be discarded in silence, leaving whoever fills the input channel blocked forever.
	if concurrency < 1 {
		concurrency = 1
	}

	var wg sync.WaitGroup

	for range concurrency {
		wg.Go(func() {
			for {
				select {
				case <-ctx.Done():
					return

				case item, ok := <-input:
					if !ok {
						return
					}

					select {
					case output <- fn(item):
					case <-ctx.Done():
						return
					}
				}
			}
		})
	}

	// Close output once all workers are done
	go func() {
		wg.Wait()
		close(output)
	}()

	return output
}
