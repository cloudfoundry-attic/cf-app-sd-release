package main

import (
	"os"
	"os/signal"
	"syscall"
	"fmt"
)

func main() {
	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, syscall.SIGTERM, os.Interrupt)

	fmt.Println("Server Started")
	select {
	case <-signalChannel:
		fmt.Println("Shutting service-discovery-controller down")
		return
	}
}
