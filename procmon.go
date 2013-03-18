//
// monitoring & restarting persistent (control master) ssh connections
//

package main

import (
    "log"
    //"fmt"
    "os/exec"
)

type Logger struct {
    prefix string
    l *log.Logger
}

// Write a message to log
func (l *Logger) Write(b []byte) (n int, err error) {
    buf := b
    if buf[len(b)-1] == '\n' {
        buf = b[:len(b)-1]
    }

    l.l.Printf("%s: %s", l.prefix, buf)
    return len(b), nil
}

func procmon(logger *log.Logger, ident string, command []string) {
    c := exec.Command(command[0], command[1:]...)
    c.Stdout = &Logger{ident + ".stdout", logger}
    c.Stderr = &Logger{ident + ".stderr", logger}

    c.Run()
}
