package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strings"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

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

	go func() {
		for {
			br := bufio.NewReader(stderr)
			line, _, err := br.ReadLine()

			if err != nil {
				log.Printf("Failed to read: %v", err)
				break
			}

			log.Printf("Error %s", line)
		}
		return
	}()

	go func() {
		for {
			br := bufio.NewReader(stdout)
			buf, err := br.Peek(4)

			if err != nil {
				log.Printf("Failed to read: %v", err)
				break
			}

			log.Printf("Stdout %v", buf)

			break
		}
		return
	}()

	err = session.Wait()
	if err != nil {
		return fmt.Errorf("Command failed: %v", err)
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
