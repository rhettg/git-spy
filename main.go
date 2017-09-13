package main

import (
	"log"
	"net"

	"github.com/rhettg/git-spy/gitspy"
)

func main() {
	config := gitspy.NewSSHServerConfig()

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
		go gitspy.HandleConnection(nConn, config)
	}
}
