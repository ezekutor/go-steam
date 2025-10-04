//go:build example
// +build example

package main

import (
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	steam "github.com/Philipp15b/go-steam/v3"
)

func main() {
	client := steam.NewClient()

	shutdown := make(chan struct{})
	eventLoopDone := make(chan struct{})
	go func() {
		defer close(eventLoopDone)
		runEventLoop(client, shutdown)
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	var cleanupOnce sync.Once
	cleanup := func(reason string) {
		cleanupOnce.Do(func() {
			log.Printf("Shutting down (%s)", reason)
			client.Disconnect()
			close(shutdown)
		})
	}

	go func() {
		if sig, ok := <-sigCh; ok {
			cleanup(sig.String())
		} else {
			cleanup("signal channel closed")
		}
		signal.Stop(sigCh)
		close(sigCh)
	}()

	waitForShutdown(sigCh)
	cleanup("main exit")
	<-eventLoopDone
	log.Println("Shutdown complete")
}

func runEventLoop(client *steam.Client, stop <-chan struct{}) {
	events := client.Events()
	for {
		select {
		case <-stop:
			return
		case event := <-events:
			handleEvent(event)
		}
	}
}

func handleEvent(event interface{}) {
	if event == nil {
		return
	}

	switch e := event.(type) {
	case steam.FatalErrorEvent:
		log.Printf("Fatal error event: %v", e)
	case *steam.DisconnectedEvent:
		log.Print("Disconnected from Steam")
	default:
		log.Printf("Event received: %T", event)
	}
}

func waitForShutdown(sigCh <-chan os.Signal) {
	for range sigCh {
	}
}
