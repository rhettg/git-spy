package gitspy

import (
	"bytes"
	"testing"
)

func TestWritePktLine(t *testing.T) {
	w := &bytes.Buffer{}

	d := []byte("Hello World")

	n, err := WritePktLine(w, d)
	if err != nil {
		t.Errorf("Got an error: %v", err)
	}

	if n != 4+len(d) {
		t.Errorf("Wrong size %d vs %d", n, 4+len(d))
	}
}

func TestWRitePktLineEmtpy(t *testing.T) {
	w := &bytes.Buffer{}

	d := []byte("")

	n, err := WritePktLine(w, d)
	if err != nil {
		t.Errorf("Got an error: %v", err)
	}

	if n != 4 {
		t.Errorf("Wrong size %d vs %d", n, 4+len(d))
	}

	if bytes.Compare(w.Bytes(), []byte("0004")) != 0 {
		t.Errorf("Wrong encoding: %v", w.Bytes())
	}
}

func TestWritePktLineFlush(t *testing.T) {
	w := &bytes.Buffer{}

	n, err := WritePktLineFlush(w)

	if err != nil {
		t.Errorf("Error: %v", err)
	}
	if bytes.Compare(w.Bytes(), []byte("0000")) != 0 {
		t.Errorf("Wrong response")
	}

	if n != 4 {
		t.Errorf("Wrong length")
	}
}
