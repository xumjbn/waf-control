package main

import (
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	slog.Info("waf-agent starting")

	// TODO: gRPC client connection to management server
	// TODO: heartbeat goroutine
	// TODO: system monitor goroutine

	fmt.Println("waf-agent is running (placeholder)")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("waf-agent shutting down")
}
