package main

import (
	"bytes"
	"io"
	"testing"
)

func Test_SmallReads(t *testing.T) {
	br := bytes.NewBufferString("hello world")
	r := RemoveComments(br)

	checkReadN(t, r, 4, "hell")
	checkReadN(t, r, 4, "o wo")
	checkReadN(t, r, 4, "rld")
	checkEmpty(t, r)
}

func Test_MultiLines(t *testing.T) {
	br := bytes.NewBufferString("one\ntwo\nthree")
	r := RemoveComments(br)

	checkRead(t, r, "one\n")
	checkRead(t, r, "two\n")
	checkRead(t, r, "three")
	checkEmpty(t, r)
}

func Test_MultiLines2(t *testing.T) {
	br := bytes.NewBufferString("one\n\nthree\n\n\n")
	r := RemoveComments(br)

	checkRead(t, r, "one\n")
	checkRead(t, r, "\n")
	checkRead(t, r, "three\n")
	checkRead(t, r, "\n")
	checkRead(t, r, "\n")
	checkEmpty(t, r)
}

func Test_Comments1(t *testing.T) {
	br := bytes.NewBufferString("one\n# comment\ntwo")
	r := RemoveComments(br)

	checkRead(t, r, "one\n")
	checkRead(t, r, "two")
	checkEmpty(t, r)
}

func Test_Comments2(t *testing.T) {
	br := bytes.NewBufferString("one\n# comment")
	r := RemoveComments(br)

	checkRead(t, r, "one\n")
	checkEmpty(t, r)
}

func Test_Comments3(t *testing.T) {
	br := bytes.NewBufferString("one\n# comment\n   \n  \t#comment\ntwo")
	r := RemoveComments(br)

	checkRead(t, r, "one\n")
	checkRead(t, r, "   \n")
	checkRead(t, r, "two")
	checkEmpty(t, r)
}

func Test_Comments4(t *testing.T) {
	br := bytes.NewBufferString("one\n// comment\n   \n  \t//comment\ntwo")
	r := RemoveComments(br)

	checkRead(t, r, "one\n")
	checkRead(t, r, "   \n")
	checkRead(t, r, "two")
	checkEmpty(t, r)
}

func Test_Comments5(t *testing.T) {
	br := bytes.NewBufferString("one\n// comment\n   \n  \t/comment\ntwo")
	r := RemoveComments(br)

	checkRead(t, r, "one\n")
	checkRead(t, r, "   \n")
	checkRead(t, r, "  \t/comment\n")
	checkRead(t, r, "two")
	checkEmpty(t, r)
}

func checkReadN(t *testing.T, rd io.Reader, bn int, exp string) {
	buf := make([]byte, bn)
	n, e := rd.Read(buf)
	if e != nil || n != len(exp) || string(buf[:n]) != exp {
		t.Errorf("Incorrect read: %q != %q, err=%v", buf[:n], exp, e)
	}
}

func checkRead(t *testing.T, rd io.Reader, exp string) {
	buf := make([]byte, 100)
	n, e := rd.Read(buf)
	if e != nil || n != len(exp) || string(buf[:n]) != exp {
		t.Errorf("Incorrect read: %q != %q, err=%v", buf[:n], exp, e)
	}
}

func checkEmpty(t *testing.T, rd io.Reader) {
	buf := make([]byte, 100)
	n, e := rd.Read(buf)
	if e == nil || n != 0 {
		t.Errorf("Incorrect EOF read: err=%v   n=%d", e, n)
	}
}
