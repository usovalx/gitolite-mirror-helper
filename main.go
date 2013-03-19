package main

import (
	"fmt"
	"log"
	"os"
	_ "os/signal"
	"time"
)

var mainLogger = log.New(os.Stderr, "", log.LstdFlags)

func main() {
	fmt.Printf("Starting\n")

	a := &ProcMonConf{
		Host:    "repo-ln",
		PreCmd:  "ssh -O exit %host",
		PreArgs: "",
		Cmd:     "echo ssh -M %host",
		Args:    ""}

	stopCh := make(chan bool)
	resCh := ProcMon(stopCh, mainLogger, a.Host, a)

	// FIXME: signal handling to shutdown child processes
	// HUP -> reload config & restart
	// INT -> shutdown
	// TERM -> shutdown
	sigCh := make(chan os.Signal, 10)
	//signal.Notify(sigCh, os.Interrupt)
	go func() {
		for {
			s := <-sigCh
			mainLogger.Printf("Signal: %d", s)
		}
	}()

	time.Sleep(100 * time.Second)
	stopCh <- true
	_ = <-resCh
}
