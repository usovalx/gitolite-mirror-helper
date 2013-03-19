package main

import (
	"fmt"
	"log"
	"os"
        "os/signal"
	"time"
)

var mainLogger = log.New(os.Stderr, "", log.LstdFlags)

func main() {
	fmt.Printf("Starting\n")

	stopCh := make(chan bool)
	resCh := ProcMon(stopCh, mainLogger, "echo", []string{"ssh", "-N", "repo-au"})

        // FIXME: signal handling to shutdown child processes
        // HUP -> reload config & restart
        // INT -> shutdown
        // TERM -> shutdown
        sigCh := make(chan os.Signal, 10)
        signal.Notify(sigCh, os.Interrupt)
        go func() {
            for {
                s := <-sigCh
                mainLogger.Printf("Signal: %d", s)
            }
        } ()

	time.Sleep(100 * time.Second)
	stopCh <- true
	_ = <-resCh
}
