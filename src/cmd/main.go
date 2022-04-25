package main

import (
	"context"
	"fmt"
	"github.com/dmalykh/refurbednotifier/cmd/config"
	"github.com/dmalykh/refurbednotifier/notifier"
	"github.com/dmalykh/refurbedsender"
	"github.com/dmalykh/refurbedsender/gate/http"
	"github.com/dmalykh/refurbedsender/middleware"
	"github.com/dmalykh/refurbedsender/queue/list"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	// Get config from flags
	var c = new(config.Config)
	if cancel, err := c.ParseFlags(); cancel != nil {
		if err != nil {
			fmt.Println(err.Error())
		}
		cancel()
		os.Exit(1)
	}

	// Graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan os.Signal)
	go func() {
		<-done
		cancel()
	}()
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	//Recover
	defer func() {
		if r := recover(); r != nil {
			cancel()
			os.Exit(1)
		}
	}()

	// Build notifier
	var queue = list.NewListQueue()
	gate, err := http.NewHTTPGate(&http.Config{
		URL:     c.URL,
		Timeout: c.Timeout,
	})
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	var s = refurbedsender.NewSender(queue, gate, false)
	go func() {
		if err := s.Run(ctx, middleware.WithThrottlingMiddleware(c.Rps, 1, time.Duration(float64(time.Second)/float64(c.Rps)))); err != nil {
			log.Fatal(err)
		}
	}()

	// Print errors to stderr
	go func() {
		for err := range s.Errors() {
			printErr(err)
		}
	}()

	// Run notifier
	var n = notifier.NewNotifier(s, c.MaxQueueSize, c.Interval)
	n.Run(ctx, os.Stdin, func(ctx context.Context, message []byte) error {
		if err := n.Send(ctx, message); err != nil {
			printErr(err)
		}

		return nil
	})
}

// Print error to stderr
func printErr(err error) {
	if _, e := os.Stderr.WriteString(err.Error() + "\n"); e != nil {
		fmt.Println(err.Error(), e.Error())
	}
}
