package main

import (
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"log"
	"net"
	"strings"

	"k8s-ssh-server/k8s"

	cryptoSSH "golang.org/x/crypto/ssh"
)

func generateHostKey() (cryptoSSH.Signer, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("failed to generate private key: %w", err)
	}

	signer, err := cryptoSSH.NewSignerFromKey(privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create signer: %w", err)
	}

	return signer, nil
}

func main() {
	k8s.InitK8sClient("/Users/antraxmin/.kube/config")

	config := &cryptoSSH.ServerConfig{
		PasswordCallback: func(c cryptoSSH.ConnMetadata, pass []byte) (*cryptoSSH.Permissions, error) {
			if c.User() == "admin" && string(pass) == "password" {
				return nil, nil
			}
			return nil, fmt.Errorf("password rejected for %q", c.User())
		},
	}

	hostKey, err := generateHostKey()
	if err != nil {
		log.Fatalf("Failed to generate host key: %v", err)
	}
	config.AddHostKey(hostKey)

	address := "0.0.0.0:2222"
	listener, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("Failed to start SSH server: %v", err)
	}
	defer listener.Close()

	log.Printf("SSH server listening on %s", address)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Failed to accept connection: %v", err)
			continue
		}

		go func(conn net.Conn) {
			defer conn.Close()
			sshConn, chans, reqs, err := cryptoSSH.NewServerConn(conn, config)
			if err != nil {
				log.Printf("Failed to handshake: %v", err)
				return
			}

			username := sshConn.User()
			log.Printf("New SSH connection from %s - user: %s", sshConn.RemoteAddr(), username)

			podName, err := k8s.CreateUserPod(strings.ToLower(username))
			if err != nil {
				log.Printf("Failed to create pod for user %s: %v", username, err)
				return
			}
			log.Printf("Pod %s created for user %s", podName, username)

			go cryptoSSH.DiscardRequests(reqs)
			for newChannel := range chans {
				if newChannel.ChannelType() != "session" {
					newChannel.Reject(cryptoSSH.UnknownChannelType, "unsupported channel type")
					continue
				}
				channel, _, err := newChannel.Accept()
				if err != nil {
					log.Printf("Failed to accept channel: %v", err)
					continue
				}

				channel.Write([]byte(fmt.Sprintf("Welcome to the SSH server, %s!\n", username)))
				channel.Write([]byte(fmt.Sprintf("Your Kubernetes pod '%s' is ready.\n", podName)))
				channel.Close()
			}
		}(conn)
	}
}
