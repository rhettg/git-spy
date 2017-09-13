package gitspy

import (
	"bytes"
	"fmt"
	"io"
)

// Write buffer to the specified writer in pkg-line format
// https://www.kernel.org/pub/software/scm/git/docs/technical/protocol-common.html
func WritePktLine(w io.Writer, b []byte) (n int, err error) {
	l := []byte(fmt.Sprintf("%x", len(b)+4))

	if len(l) < 4 {
		l = append(bytes.Repeat([]byte("0"), 4-len(l)), l...)
	}

	// Is it better to call Write multiple times or do string formatting? b could be very big.

	n, err = w.Write(l)
	if err != nil {
		return
	}

	if n < len(l) {
		err = io.ErrShortWrite
		return
	}

	written, err := w.Write(b)
	n += written

	if err != nil {
		return
	}

	if written < len(b) {
		err = io.ErrShortWrite
		return
	}

	return
}

func WritePktLineFlush(w io.Writer) (n int, err error) {
	return w.Write([]byte("0000"))
}
