package main

import (
	"time"

	"github.com/vegidio/go-sak/async"
)

func main() {
	array := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20}

	ch := async.SliceToChannel(array, 5, func(n int) int {
		time.Sleep(1 * time.Second)
		return n * 2
	})

	for result := range ch {
		println(result)
	}
}
