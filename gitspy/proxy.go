package gitspy

import (
	"bufio"
	"fmt"
	"io"
	"log"

	"golang.org/x/crypto/ssh"
)

func proxyUploadPack(c ssh.Channel, cmd string) (err error) {
	config := &ssh.ClientConfig{
		User:            "git",
		Auth:            []ssh.AuthMethod{sshAgentAuth()},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	client, err := ssh.Dial("tcp", "github.com:22", config)
	if err != nil {
		return fmt.Errorf("Failed to connect: %v", err)
	}

	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("Failed to create new session: %v", err)
	}

	defer session.Close()

	stdin, err := session.StdinPipe()
	if err != nil {
		return fmt.Errorf("Failed to open stdin: %v", err)
	}

	stdout, err := session.StdoutPipe()
	if err != nil {
		return fmt.Errorf("Failed to open stdout: %v", err)
	}

	stderr, err := session.StderrPipe()
	if err != nil {
		return fmt.Errorf("Failed to open stderr: %v", err)
	}

	err = session.Start(cmd)
	if err != nil {
		return fmt.Errorf("Failed to start command: %v", err)
	}

	gs := NewGitSpy(c, stdin)

	go func() {
		cp := gs.ClientPipe()
		_, err := io.Copy(cp, c)
		if err != nil && err != io.EOF {
			log.Fatalf("Failed to Copy to client pipe: %v", err)
		}

		log.Printf("Client copy complete")

		cp.Close()

		return
	}()

	go func() {
		sp := gs.ServerPipe()
		_, err := io.Copy(sp, stdout)

		if err != nil && err != io.EOF {
			log.Fatalf("Failed to Copy to server pipe: %v", err)
		}

		log.Printf("Server copy complete")
		sp.Close()

		return
	}()

	serr := session.Wait()

	for {
		br := bufio.NewReader(stderr)
		line, _, err := br.ReadLine()

		if len(line) > 0 {
			log.Printf("Error: %s", line)
		}

		if err != nil {
			if err != io.EOF {
				log.Printf("Failed to read: %v", err)
			}
			break
		}
	}

	if serr != nil {
		return fmt.Errorf("Command failed: %v", serr)
	}

	return nil
}
