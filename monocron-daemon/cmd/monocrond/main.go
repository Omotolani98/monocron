package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	monocrond "xyz.tolaniverse.monocron-daemon"
)

func main() {
	addr := getenv("MONOCROND_ADDR", ":50030")
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGTERM, syscall.SIGINT)
		<-c
		cancel()
	}()
	log.Printf("monocrond starting on %s", addr)
	if err := monocrond.Serve(ctx, addr); err != nil {
		time.Sleep(100 * time.Millisecond)
		log.Fatalf("server error: %v", err)
	}
}

func getenv(k, def string) string {
	v := os.Getenv(k)
	if v == "" {
		return def
	}
	return v
}
