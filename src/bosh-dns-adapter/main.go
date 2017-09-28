package main

import (
	"fmt"
	"os"
	"syscall"
	"os/signal"
)

func main() {
	fmt.Println("bosh dns adapter has started")

	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, syscall.SIGTERM, os.Interrupt, os.Kill)

	select {
	case <-signalChannel:
		return
	}
}
