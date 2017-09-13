package main

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strings"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

type GitSpy struct {
	// Server
	s ssh.Session

	// Client
	c ssh.Channel
}

func (gs *GitSpy) Write(p []byte) (n int, err error) {
	return

}

func (gs *GitSpy) Read(p []byte) (n int, err error) {
	return

}

func passwordCallback(c ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
	//log.Printf("Checking password for %v", c)
	return nil, nil
}

func publicKeyCallback(c ssh.ConnMetadata, pubKey ssh.PublicKey) (*ssh.Permissions, error) {
	//log.Printf("Checking public key %v for %v", pubKey, c)
	return nil, nil
}

func sshAgentAuth() ssh.AuthMethod {
	sshAgent, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK"))
	if err != nil {
		log.Fatalf("Failed to open ssh agent")
	}

	return ssh.PublicKeysCallback(agent.NewClient(sshAgent).Signers)
}

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

	// If I didn't care about spying
	go func() {
		_, err := io.Copy(stdin, c)
		if err != nil && err != io.EOF {
			log.Fatalf("Failed to Copy to stdin: %v", err)
		}

		stdin.Close()

		log.Printf("Closed stdin")

		return
	}()

	go func() {
		io.Copy(c, stdout)

		if err != nil && err != io.EOF {
			log.Fatalf("Failed to Copy to channel: %v", err)
		}

		// c.Close()
		log.Printf("Channel copy complete")

		return
	}()

	/*
		for {
			br := bufio.NewReader(stdout)
			// TODO prefix
			line, _, err := br.ReadLine()

			if len(line) > 0 {
				log.Printf("Stdout %s", line)
				n, err = stdin.Write(line)
				if n != len(line) {
					log.Fatalf("TODO: partial write")
				}
				if err != nil {
					return fmt.Errorf("Failed to write")
				}
			}

			if err != nil {
				if err != io.EOF {
					log.Printf("Failed to read: %v", err)

				}
				break
			}
		}
	*/

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

func handleChannel(c ssh.Channel, r <-chan *ssh.Request) {
	for req := range r {
		log.Printf("Channel request: %s", req.Type)

		if req.Type == "exec" {
			// Parse out our payload, stripping 3 null bytes and an ENQ
			p := string(req.Payload[4:])
			if strings.HasPrefix(p, "git-upload-pack") {
				req.Reply(true, nil)

				err := proxyUploadPack(c, p)
				if err != nil {
					log.Fatalf("Failed to write to channel: %v", err)
				}

				log.Printf("Wrote reply to channel")

				// Nothing allowed after exec?
				break
			} else {
				log.Printf("Unknown exec '%s' command, failing", p)
				req.Reply(false, nil)
			}
		} else {
			req.Reply(false, nil)
		}
	}

	log.Printf("Closing channel")

	c.Close()
}

func handleConnection(c net.Conn, config *ssh.ServerConfig) {
	conn, chans, reqs, err := ssh.NewServerConn(c, config)
	if err != nil {
		log.Fatal("failed to handshake: ", err)
	}

	log.Printf("logged in")

	go ssh.DiscardRequests(reqs)

	for newChannel := range chans {
		log.Printf("New channel '%s'", newChannel.ChannelType())

		channel, requests, err := newChannel.Accept()
		if err != nil {
			log.Fatalf("Could not accept channel: %v", err)
		}

		go handleChannel(channel, requests)
	}

	conn.Close()
}

func main() {
	config := &ssh.ServerConfig{
		PasswordCallback:  passwordCallback,
		PublicKeyCallback: publicKeyCallback,
	}

	privateBytes, err := ioutil.ReadFile("id_rsa")
	if err != nil {
		log.Fatal("Failed to load private key: ", err)
	}

	private, err := ssh.ParsePrivateKey(privateBytes)
	if err != nil {
		log.Fatal("Failed to parse private key: ", err)
	}

	config.AddHostKey(private)

	listener, err := net.Listen("tcp", "127.0.0.1:2022")
	if err != nil {
		log.Fatal("failed to listen for connection: ", err)
	}

	for {
		nConn, err := listener.Accept()
		if err != nil {
			log.Fatal("failed to accept incoming connection: ", err)
		}

		log.Printf("Accepted %v", nConn.RemoteAddr())
		go handleConnection(nConn, config)
	}
}
