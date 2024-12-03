package main

import (
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"log"
	"net"
	"os"

	"k8s-ssh-server/db"
	"k8s-ssh-server/k8s"

	cryptoSSH "golang.org/x/crypto/ssh"
)

func generateHostKey() (cryptoSSH.Signer, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("failed to generate private key: %w", err)
	}
	return cryptoSSH.NewSignerFromKey(privateKey)
}

func handleChannel(channel cryptoSSH.Channel, requests <-chan *cryptoSSH.Request, username string) {
	defer channel.Close()

	for req := range requests {
		switch req.Type {
		case "shell":
			req.Reply(true, nil)
			channel.Write([]byte(fmt.Sprintf("Welcome to the SSH server, %s!\n", username)))

			namespace, podName, err := k8s.GetPodForUser(username)
			if err != nil {
				log.Printf("Failed to get pod for user %s: %v", username, err)
				return
			}

			channel.Write([]byte(fmt.Sprintf("Connected to pod %s in namespace %s\n", podName, namespace)))
		case "exec":
			req.Reply(true, nil)

			namespace, podName, err := k8s.GetPodForUser(username)
			if err != nil {
				log.Printf("Failed to get pod: %v", err)
				continue
			}

			cmd := string(req.Payload[4:])
			output, err := k8s.ExecuteCommandInPod(namespace, podName, "", cmd)
			if err != nil {
				channel.Write([]byte(fmt.Sprintf("Error: %v\n", err)))
				continue
			}
			channel.Write([]byte(output))
		default:
			req.Reply(false, nil)
		}
	}
}

func main() {
	db.InitDB()
	defer db.DB.Close()

	k8s.InitK8sClient(os.Getenv("KUBECONFIG"))

	config := &cryptoSSH.ServerConfig{
		PasswordCallback: func(c cryptoSSH.ConnMetadata, pass []byte) (*cryptoSSH.Permissions, error) {
			isAuthenticated, err := db.AuthenticateUser(c.User(), string(pass))
			if err != nil {
				log.Printf("Authentication error for user %s: %v", c.User(), err)
				return nil, fmt.Errorf("authentication error")
			}
			if !isAuthenticated {
				return nil, fmt.Errorf("invalid username or password")
			}
			return &cryptoSSH.Permissions{}, nil
		},
	}

	hostKey, err := generateHostKey()
	if err != nil {
		log.Fatalf("Failed to generate host key: %v", err)
	}
	config.AddHostKey(hostKey)

	listener, err := net.Listen("tcp", "0.0.0.0:2222")
	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
	defer listener.Close()

	log.Printf("SSH server listening on 0.0.0.0:2222")

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

			log.Printf("New SSH connection from %s - user: %s", sshConn.RemoteAddr(), sshConn.User())

			go func(reqs <-chan *cryptoSSH.Request) {
				for req := range reqs {
					if req.Type == "shell" {
						req.Reply(true, nil)
					} else {
						req.Reply(false, nil)
					}
				}
			}(reqs)

			for newChannel := range chans {
				if newChannel.ChannelType() != "session" {
					newChannel.Reject(cryptoSSH.UnknownChannelType, "unsupported channel type")
					continue
				}

				channel, requests, err := newChannel.Accept()
				if err != nil {
					log.Printf("Failed to accept channel: %v", err)
					continue
				}

				go handleChannel(channel, requests, sshConn.User())
			}
		}(conn)
	}
}
