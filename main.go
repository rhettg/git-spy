package main

import (
	"io/ioutil"
	"log"
	"net"
	"strings"

	"golang.org/x/crypto/ssh"
)

func passwordCallback(c ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
	//log.Printf("Checking password for %v", c)
	return nil, nil
}

func publicKeyCallback(c ssh.ConnMetadata, pubKey ssh.PublicKey) (*ssh.Permissions, error) {
	//log.Printf("Checking public key %v for %v", pubKey, c)
	return nil, nil
}

func handleChannel(c ssh.Channel, r <-chan *ssh.Request) {
	for req := range r {
		log.Printf("Channel request: %s", req.Type)

		if req.Type == "exec" {
			// Parse out our payload (3 null bytes and a ENQ)
			p := string(req.Payload[4:])
			if strings.HasPrefix(p, "git-upload-pack") {
				//proxyUploadPack(c)

				req.Reply(true, nil)

				_, err := c.Write([]byte("hello world\n"))
				if err != nil {
					log.Fatalf("Failed to write to channel")
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
