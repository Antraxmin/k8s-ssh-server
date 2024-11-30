package ssh

import (
	"log"
	"net"

	"golang.org/x/crypto/ssh"
)

func StartSSHServer(config *ssh.ServerConfig, address string) {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("Failed to listen on %s: %s", address, err)
	}
	log.Printf("SSH server listening on %s", address)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Failed to accept incoming connection: %s", err)
			continue
		}
		go handleConnection(conn, config)
	}
}
