package async

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestConcurrentChannelContext_ProcessesEveryItem(t *testing.T) {
	input := make(chan int)

	go func() {
		defer close(input)
		for i := 1; i <= 10; i++ {
			input <- i
		}
	}()

	output := ConcurrentChannelContext(context.Background(), input, 3, func(n int) int {
		return n * n
	})

	sum := 0
	for value := range output {
		sum += value
	}

	assert.Equal(t, 385, sum)
}

func TestConcurrentChannelContext_StopsOnCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	input := make(chan int)

	go func() {
		defer close(input)
		for i := 0; i < 1_000; i++ {
			select {
			case input <- i:
			case <-ctx.Done():
				return
			}
		}
	}()

	output := ConcurrentChannelContext(ctx, input, 3, func(n int) int { return n })

	// Consume a couple of results, then walk away
	<-output
	cancel()

	drained := make(chan struct{})
	go func() {
		defer close(drained)
		for range output {
			// Drain until the workers give up and the channel is closed
		}
	}()

	select {
	case <-drained:
	case <-time.After(5 * time.Second):
		t.Fatal("the output channel was never closed after the context was cancelled")
	}
}
