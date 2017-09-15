package gitspy

import (
	"fmt"
	"io"
	"log"
)

type GitSpy struct {
	c io.WriteCloser
	s io.WriteCloser
}

func (gs *GitSpy) ClientPipe() io.WriteCloser {
	r, w := io.Pipe()

	go func() {
		_, err := io.Copy(gs.s, r)
		if err != nil && err != io.EOF {
			r.CloseWithError(fmt.Errorf("Failed writing to server: %v", err))
			return
		}

		log.Printf("ClientPipe exited, closing server")
		gs.s.Close()

		return
	}()

	return w
}

type filterfunc func([]byte) ([]byte, error)

func logServer(b []byte) ([]byte, error) {
	log.Printf("S: %s", b)
	return b, nil
}

func proxyPktLine(dst io.Writer, src io.Reader, f filterfunc) (done bool, err error) {
	// TODO: pool this?
	b := make([]byte, 65516)

	n, err := ParsePktLine(src, b)
	if err != nil {
		if err == ErrFlushPkt {
			log.Printf("S: FLUSH")
			_, err = WritePktLineFlush(dst)
			if err != nil {
				return true, nil
			}

			return true, nil
		} else if err == io.EOF {
			log.Printf("EOF from server")
		} else {
			log.Printf("Error parsing from server: %v", err)
		}

		return true, fmt.Errorf("Failed parsing pkt: %v", err)
	}

	ob, err := f(b[0:n])

	_, err = WritePktLine(dst, ob)
	if err != nil {
		return true, fmt.Errorf("Failed writing pkt: %v", err)
	}

	return false, nil
}

func (gs *GitSpy) ServerPipe() io.WriteCloser {
	r, w := io.Pipe()

	go func() {
		var err error

		done := false
		for !done {
			done, err = proxyPktLine(gs.c, r, logServer)
			if err != nil {
				r.CloseWithError(fmt.Errorf("Failed proxying to client: %v", err))
				break
			}
		}

		if err != nil {
			r.CloseWithError(fmt.Errorf("Failed writing from server to client: %v", err))
			return
		} else {
			_, err = io.Copy(gs.c, r)
			if err != nil {
				r.CloseWithError(fmt.Errorf("Failed direct writing to client: %v", err))
				return
			}
		}

		log.Printf("ServerPipe exited normally")
		return
	}()

	return w
}

func (gs *GitSpy) Close() {
	gs.c.Close()
	gs.s.Close()
}

func NewGitSpy(client io.WriteCloser, server io.WriteCloser) *GitSpy {
	gs := GitSpy{client, server}

	return &gs
}
