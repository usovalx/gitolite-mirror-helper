//
// monitoring & restarting persistent (control master) ssh connections
//

package main

import (
	"os"
	"os/exec"
	"strings"
	"time"
)

type ProcMonConf struct {
	Host    string
	PreCmd  string
	PreArgs string
	Cmd     string
	Args    string
}

const (
	minDelay          = 1 * time.Second
	maxDelay          = 3 * time.Minute
	goodTimeThreshold = 30 * time.Second
)

func ProcMonRun(dieCh <-chan bool,
	ident string, cnf *ProcMonConf) <-chan bool {
	c := make(chan bool, 1)
	go procMonMain(dieCh, c, ident, cnf)
	return c
}

func procMonMain(dieCh <-chan bool, doneCh chan<- bool,
	ident string, cnf *ProcMonConf) {
	defer func() { doneCh <- true }()

	// construct & split commands
	preCmdArr := splitCommand(cnf.PreCmd, cnf.PreArgs, cnf.Host)
	cmdArr := splitCommand(cnf.Cmd, cnf.Args, cnf.Host)

	var stop bool
	var cmd *exec.Cmd
	var slaveDied chan bool

	// First run pre-command
	if len(preCmdArr) > 0 {
		stop = false
		logger.Printf("%s: starting %v", ident+".pre", preCmdArr)
		cmd, slaveDied = startSlave(ident+".pre", preCmdArr)
		waitForPreCmd:
		select {
		case _ = <-dieCh:
			logger.Printf("%s.pre: exit request: killing slave", ident)
			stop = true
			cmd.Process.Signal(os.Interrupt)
			goto waitForPreCmd
		case _ = <-slaveDied:
			if stop {
				return
			}
		}
	}

	// And then command itself
	stop = false
	startDelay := time.Duration(0)
	for !stop {
		// part 1 -- starting new slave process
		t := time.NewTimer(startDelay)
		select {
		case _ = <-dieCh:
			t.Stop()
			return
		case _ = <-t.C:
			logger.Printf("%s: starting %v", ident, cmdArr)
			cmd, slaveDied = startSlave(ident, cmdArr)
		}

	waitForIt:
		// part 2 -- monitoring running slave process
		select {
		case _ = <-dieCh:
			logger.Printf("%s: exit request: killing slave", ident)
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
			logger.Printf("%s: slave died, restarting in %g second(s)", ident, float64(startDelay)/float64(time.Second))
			cmd = nil
		}
	}
	return
}

func startSlave(ident string, args []string) (cmd *exec.Cmd, ch chan bool) {
	cmd = exec.Command(args[0], args[1:]...)
	cmd.Stdin = nil
	cmd.Stdout = &LogWriter{ident + ".stdout"}
	cmd.Stderr = &LogWriter{ident + ".stderr"}

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

func splitCommand(cmd, args, host string) []string {
	s := strings.Replace(cmd, "%args", args, -1)
	s = strings.Replace(s, "%host", host, -1)
	return strings.Fields(s)
}

type LogWriter struct {
	ident string
}

func (l *LogWriter) Write(b []byte) (n int, err error) {
	n = len(b)
	if b[n-1] == '\n' {
		b = b[:n-1]
	}

	logger.Printf("%s: %s", l.ident, b)
	return n, nil
}
