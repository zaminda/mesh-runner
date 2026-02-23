package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/nats-io/nats.go"
	"github.com/zaminda/mesh-runner/internal"
)

func main() {

	nc, _ := nats.Connect("nats://localhost:4222")

	defer nc.Close()

	nc.Subscribe(internal.RequestNATSSubject, func(msg *nats.Msg) {
		fmt.Println("received message:", string(msg.Data))
	})

	log.Println("worker started, waiting for jobs...")

	// Wait for SIGTERM or SIGINT
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	log.Println("shutting down worker...")
	log.Println("worker stopped")
}
