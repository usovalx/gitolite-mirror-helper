package main

import (
	"flag"
	"log"
	"os"
	_ "os/signal"
	"time"
)

type Config struct {
	ProcMonHosts []string
	ProcMon      *ProcMonConf
}

type ProcMonCtl struct {
	stopCh chan<- bool
	diedCh <-chan bool
}

var logger = log.New(os.Stderr, "", log.LstdFlags)
var configName = flag.String("c", "", "config file")

func main() {
	flag.Parse()

	if *configName == "" {
		logger.Fatal("No config")
	}

	cnf, err := ReadConfig(*configName)
	if err != nil {
		logger.Fatalf("Error reading config: %v", err)
	}
	if err := CheckConfig(cnf); err != nil {
		logger.Fatalf("Invalid config: %v", err)
	}
	logger.Printf("Config loaded: %#v", cnf)

	// start all processes
	procMons := make(map[string]ProcMonCtl)
	for _, h := range cnf.ProcMonHosts {
		c := *cnf.ProcMon
		c.Host = h
		stopCh := make(chan bool, 1)
		resCh := ProcMonRun(stopCh, c.Host, &c)
		procMons[h] = ProcMonCtl{stopCh, resCh}
	}

	// FIXME: signal handling to shutdown child processes
	// HUP -> reload config & restart
	// INT -> shutdown
	// TERM -> shutdown
	sigCh := make(chan os.Signal, 10)
	//signal.Notify(sigCh, os.Interrupt)
	go func() {
		for {
			s := <-sigCh
			logger.Printf("Signal: %d", s)
		}
	}()

	time.Sleep(100 * time.Second)
	for _, ctl := range procMons {
		ctl.stopCh <- true
	}
	for _, ctl := range procMons {
		_ = <-ctl.diedCh
	}
}
