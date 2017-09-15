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

func (gs *GitSpy) ServerPipe() io.WriteCloser {
	r, w := io.Pipe()

	go func() {
		b := make([]byte, 65516)
		for {
			n, err := ParsePktLine(r, b)
			if err != nil && err != ErrFlushPkt {
				log.Printf("Error parsing from server: %v", err)
				if err == io.EOF {
					log.Printf("EOF from server")
				} else {
					r.CloseWithError(fmt.Errorf("Failed parsing from server: %v", err))
				}

				break
			}

			if err == ErrFlushPkt {
				log.Printf("S: FLUSH")
				_, err = WritePktLineFlush(gs.c)
				if err != nil {
					r.CloseWithError(fmt.Errorf("Failed writing to client: %v", err))
					break
				}
				// Now that we've flushed, just send the rest direct
				_, err := io.Copy(gs.c, r)
				if err != nil {
					r.CloseWithError(fmt.Errorf("Failed writing to client: %v", err))
				}
				break

			} else {
				log.Printf("S: %s", b[0:n])

				_, err = WritePktLine(gs.c, b[0:n])
				if err != nil {
					log.Printf("EOF writing to client")
					r.CloseWithError(fmt.Errorf("Failed writing to client: %v", err))
					break
				}
			}
		}

		//_, err := io.Copy(gs.c, r)
		log.Printf("ServerPipe exited, closing client")
		gs.c.Close()

		return
	}()

	return w
}

func NewGitSpy(client io.WriteCloser, server io.WriteCloser) *GitSpy {
	gs := GitSpy{client, server}

	return &gs
}
