package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
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

// host -> ProcMonCtl
var procMons = make(map[string]ProcMonCtl)

func main() {
	flag.Parse()

	if *configName == "" {
		usage()
	}

	cnf, err := ReadConfig(*configName)
	if err != nil {
		logger.Fatalf("Reading config: %v", err)
	}
	cnfSt, err := os.Stat(*configName)
	if err != nil {
		logger.Fatalf("Reading config: %v", err)
	}
	logger.Printf("Config loaded: %#v", cnf)

	// install signal handlers
	dieSigCh := make(chan os.Signal, 10)
	signal.Notify(dieSigCh, os.Interrupt, syscall.SIGTERM)
	reloadSigCh := make(chan os.Signal, 10)
	signal.Notify(reloadSigCh, syscall.SIGHUP)

	// start all monitored processes
	procMons = startSlaves(cnf)

	stopped := make(chan bool)
	checkConfig := time.Tick(30 * time.Second)
	stopInProgress := false
	reloadInProgress := false
	for {
		select {
		case sig := <-dieSigCh:
			logger.Printf("Signal %d: shutting down", sig)
			if !stopInProgress {
				stopInProgress = true
				go stopSlaves(stopped)
			}

		case _ = <-stopped:
			if stopInProgress {
				os.Exit(0)
			} else {
				if !reloadInProgress { // unreachable
					panic("Stopped when not reloading or shutting down?")
				}
				reloadInProgress = false
				procMons = startSlaves(cnf)
			}

		case sig := <-reloadSigCh:
			if stopInProgress {
				logger.Printf("Signal %d: can't reload -- shutting down", sig)
			} else if reloadInProgress {
				logger.Printf("Signal %d: can't reload -- already doing it", sig)
			} else {
				logger.Printf("Signal %d: reloading", sig)
				if newCnf := tryReload(*configName, cnf); newCnf != nil {
					// FIXME: the most stupid approach to restarting
					// stop everything and start new ones instead
					reloadInProgress = true
					cnf = newCnf
					go stopSlaves(stopped)
				}
			}

		case _ = <-checkConfig:
			if reloadInProgress || stopInProgress {
				continue
			}
			st, err := os.Stat(*configName)
			if err != nil {
				logger.Printf("WARN: Can't stat config %q", *configName)
				continue
			}
			if st.Size() != cnfSt.Size() || st.ModTime() != cnfSt.ModTime() {
				logger.Printf("It seems config has changed, reloading it")
				cnfSt = st
				if newCnf := tryReload(*configName, cnf); newCnf != nil {
					// FIXME: the most stupid approach to restarting
					// stop everything and start new ones instead
					reloadInProgress = true
					cnf = newCnf
					go stopSlaves(stopped)
				}
			}
		}
	}
}

func tryReload(name string, old *Config) *Config {
	newCnf, err := ReadConfig(*configName)
	if err != nil {
		// NOT fatal error -- just ignore new config
		logger.Printf("Reloading config: %v", err)
		return nil
	}
	if ConfigEqual(newCnf, old) {
		logger.Printf("Config hasn't changed")
		return nil
	}
	return newCnf
}

func startSlaves(cnf *Config) map[string]ProcMonCtl {
	handles := make(map[string]ProcMonCtl)
	for _, h := range cnf.ProcMonHosts {
		c := *cnf.ProcMon
		c.Host = h
		stopCh := make(chan bool, 1)
		resCh := ProcMonRun(stopCh, c.Host, &c)
		handles[h] = ProcMonCtl{stopCh, resCh}
	}
	return handles
}

func stopSlaves(done chan<- bool) {
	for _, ctl := range procMons {
		ctl.stopCh <- true
	}
	for _, ctl := range procMons {
		_ = <-ctl.diedCh
	}
	done <- true
}

func usage() {
	usage := `Usage: gitolite-mirror-helper [flags]
some more stuff
`
	fmt.Fprint(os.Stderr, usage)
	os.Exit(1)
}
