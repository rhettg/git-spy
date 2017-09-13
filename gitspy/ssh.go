package gitspy

import (
	"io/ioutil"
	"log"

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

func NewSSHServerConfig() *ssh.ServerConfig {
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

	return config
}
