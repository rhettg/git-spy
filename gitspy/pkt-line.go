package gitspy

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
)

var ErrFlushPkt = errors.New("flush packet")

var ErrInvalidSize = errors.New("invalid size")

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

func ParsePktLine(r io.Reader, b []byte) (n int, err error) {
	sb := make([]byte, 4)
	n, err = r.Read(sb)
	if err != nil {
		return
	}

	if n != 4 {
		return n, fmt.Errorf("Wrong characters read")
	}

	db, err := hex.DecodeString(string(sb))
	if err != nil {
		return
	}

	hbs := binary.BigEndian.Uint16(db)

	if hbs == 0 {
		return 0, ErrFlushPkt
	} else if hbs <= 4 {
		return 0, ErrInvalidSize
	}

	bs := int(hbs) - 4

	if len(b) > int(bs) {
		b = b[0:bs]
	}

	n, err = io.ReadFull(r, b)
	if err != nil {
		if err == io.EOF {
			err = nil
		} else {
			return
		}
	}

	if n < int(bs) {
		err = io.ErrShortBuffer
	}

	return n, err
}
