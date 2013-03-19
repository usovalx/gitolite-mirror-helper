package main

import (
	"fmt"
	"log"
	"os"
	"time"
)

var mainLogger = log.New(os.Stderr, "", log.LstdFlags)

func main() {
	fmt.Printf("Starting\n")

	stopCh := make(chan bool)
	resCh := ProcMon(stopCh, mainLogger, "echo", []string{"false", "-N", "repo-au"})

	time.Sleep(10 * time.Second)
	stopCh <- true
	_ = <-resCh
}
