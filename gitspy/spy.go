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
		_, err := io.Copy(gs.c, r)
		if err != nil && err != io.EOF {
			r.CloseWithError(fmt.Errorf("Failed writing to client: %v", err))
			return
		}

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
