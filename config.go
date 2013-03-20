package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

func ConfigEqual(c1, c2 *Config) bool {
	r1 := (len(c1.ProcMonHosts) == len(c2.ProcMonHosts)) && (*c1.ProcMon == *c2.ProcMon)
	if r1 {
		for i := 0; r1 && i < len(c1.ProcMonHosts); i++ {
			r1 = r1 && c1.ProcMonHosts[i] == c2.ProcMonHosts[i]
		}
	}

	return r1
}

func ReadConfig(fname string) (c *Config, e error) {
	f, e := os.Open(fname)
	if e != nil {
		return
	}
	defer f.Close()

	c = &Config{}
	e = json.NewDecoder(RemoveComments(f)).Decode(c)
	if e != nil {
		return
	}
	e = checkConfig(c)
	return
}

func checkConfig(c *Config) error {
	// FIXME: ProcMon is the only stuff we do so far,
	// so it must be configured
	if len(c.ProcMonHosts) == 0 {
		return fmt.Errorf("ProcMonHosts list is empty -- nothing to do")
	}
	if c.ProcMon == nil || strings.TrimSpace(c.ProcMon.Cmd) == "" {
		return fmt.Errorf("Cmd in ProcMon config is empty -- nothing to do")
	}
	return nil
}

func RemoveComments(r io.Reader) io.Reader {
	return &Uncommenter{br: bufio.NewReader(r)}
}

type Uncommenter struct {
	br   *bufio.Reader
	last []byte
}

func (u *Uncommenter) Read(buf []byte) (n int, e error) {
start:
	if len(u.last) > 0 {
		n = copy(buf, u.last)
		u.last = u.last[n:]
		return n, nil
	}

nextLine:
	b, e := u.br.ReadBytes('\n')
	// drop comment lines
	bt := bytes.TrimLeft(b, " \t")
	if (len(bt) > 0 && string(bt[:1]) == "#") || (len(bt) > 1 && string(bt[:2]) == "//") {
		goto nextLine
	}
	u.last = b
	if len(b) > 0 {
		goto start
	}
	return 0, e
}
