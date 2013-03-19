//
// monitoring & restarting persistent (control master) ssh connections
//

package main

import (
	"log"
	"os"
	"os/exec"
	"time"
)

const (
	minDelay          = 1 * time.Second
	maxDelay          = 3 * time.Minute
	goodTimeThreshold = 30 * time.Second
)

func ProcMon(dieCh <-chan bool, logger *log.Logger, ident string, args []string) <-chan bool {
	c := make(chan bool, 1)
	go procMonLoop(dieCh, c, logger, ident, args)
	return c
}

func procMonLoop(dieCh <-chan bool, doneCh chan<- bool, logger *log.Logger,
	ident string, args []string) {

	defer func() { doneCh <- true }()

	stop := false
	startDelay := time.Duration(0)
	slaveDied := make(chan bool, 1)
	cmd := (*exec.Cmd)(nil)
	for !stop {
		// part 1 -- starting new slave process
		t := time.NewTimer(startDelay)
		select {
		case _ = <-dieCh:
			t.Stop()
			return
		case _ = <-t.C:
			logger.Printf("%s: starting %s", ident, args)
			cmd, slaveDied = startSlave(logger, ident, args)
		}

	waitForIt:
		// part 2 -- monitoring running slave process
		select {
		case _ = <-dieCh:
			stop = true
			cmd.Process.Signal(os.Interrupt)
			goto waitForIt
		case r := <-slaveDied:
			if r {
				startDelay = minDelay
			} else {
				startDelay *= 2 // exponential back-off
				if startDelay < minDelay {
					startDelay = minDelay
				}
				if startDelay > maxDelay {
					startDelay = maxDelay
				}
			}
			logger.Printf("%s: slave died, restarting in %8.3f seconds", ident, float64(startDelay)/float64(time.Second))
			cmd = nil
		}
	}
	return
}

func startSlave(logger *log.Logger, ident string, args []string) (cmd *exec.Cmd, ch chan bool) {
	cmd = exec.Command(args[0], args[1:]...)
	cmd.Stdin = nil
	cmd.Stdout = &LogWriter{ident + ".stdout", logger}
	cmd.Stderr = &LogWriter{ident + ".stderr", logger}

	ch = make(chan bool, 1)
	err := cmd.Start()
	if err != nil {
		logger.Printf("%s: start: %s", ident, err)
		ch <- false
	} else {
		go func() {
			startTime := time.Now()
			err := cmd.Wait()
			if err != nil {
				logger.Printf("%s: wait: %s", ident, err)
			}
			// FIXME: stats?
			//logger.Printf("%s: finished: %s", ident, cmd.ProcessState)
			ch <- (time.Since(startTime) > goodTimeThreshold)
		}()
	}
	return
}

type LogWriter struct {
	ident  string
	logger *log.Logger
}

func (l *LogWriter) Write(b []byte) (n int, err error) {
	n = len(b)
	if b[n-1] == '\n' {
		b = b[:n-1]
	}

	l.logger.Printf("%s: %s", l.ident, b)
	return n, nil
}
