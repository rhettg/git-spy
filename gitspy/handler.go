package gitspy

import (
	"log"
	"net"
	"os"
	"strings"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

func sshAgentAuth() ssh.AuthMethod {
	sshAgent, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK"))
	if err != nil {
		log.Fatalf("Failed to open ssh agent")
	}

	return ssh.PublicKeysCallback(agent.NewClient(sshAgent).Signers)
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

func HandleConnection(c net.Conn, config *ssh.ServerConfig) {
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
