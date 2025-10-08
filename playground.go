package main

import (
	"fmt"
	"time"

	"github.com/vegidio/go-sak/async"
)

func main() {
	in := make(chan int)

	// Producer
	go func() {
		for i := 1; i <= 50; i++ {
			in <- i
		}
		close(in)
	}()

	out := async.ProcessChannel(in, 5, func(n int) string {
		time.Sleep(3 * time.Second) // simulate work
		return fmt.Sprintf("Time %s, result %d", time.Now(), n)
	})

	for result := range out {
		fmt.Println(result)
	}
}
