package main

import (
	"fmt"
	"time"
)

func main() {
	start := time.Now()

	// Simulate an operation
	time.Sleep(25 * time.Millisecond)

	elapsed := time.Since(start)

	fmt.Printf("Elapsed: %v\n", elapsed)
	fmt.Printf(
		"Milliseconds: %.3f ms\n",
		elapsed.Seconds()*1000,
	)
}
