package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	fmt.Println("Waiting for SIGINT or SIGTERM...")

	<-ctx.Done()

	fmt.Println("Shutdown signal received")
}
