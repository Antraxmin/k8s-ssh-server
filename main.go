package main

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"sync"

	"k8s-ssh-server/db"
	"k8s-ssh-server/k8s"
	"k8s-ssh-server/ssh"

	cryptoSSH "golang.org/x/crypto/ssh"
)

var hostKey crypto.Signer

func getOrCreateHostKey() (cryptoSSH.Signer, error) {
	keyBytes, err := k8s.GetHostKey()
	if err == nil {
		signer, err := cryptoSSH.ParsePrivateKey(keyBytes)
		if err == nil {
			return signer, nil
		}
	}

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	err = k8s.SaveHostKey(privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to save host key: %v", err)
	}

	return cryptoSSH.NewSignerFromKey(privateKey)
}

func handleShell(channel cryptoSSH.Channel, requests <-chan *cryptoSSH.Request, username string) {
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		for req := range requests {
			switch req.Type {
			case "pty-req":
				req.Reply(true, nil)
			case "shell":
				req.Reply(true, nil)
				channel.Write([]byte(fmt.Sprintf("Welcome to the SSH server, %s!\n", username)))

				namespace, podName, err := k8s.GetPodForUser(username)
				if err != nil {
					log.Printf("Failed to get pod for user %s: %v", username, err)
					return
				}

				channel.Write([]byte(fmt.Sprintf("Connected to pod %s in namespace %s\n", podName, namespace)))

				go io.Copy(channel, channel.Stderr())
				io.Copy(channel.Stderr(), channel)
			case "window-change":
				req.Reply(true, nil)
			default:
				req.Reply(true, nil)
			}
		}
	}()

	wg.Wait()
}

func handleExec(channel cryptoSSH.Channel, req *cryptoSSH.Request, username string) {
	cmd := string(req.Payload[4:])
	namespace, podName, err := k8s.GetPodForUser(username)
	if err != nil {
		log.Printf("Failed to get pod: %v", err)
		channel.Write([]byte(fmt.Sprintf("Error: %v\n", err)))
		return
	}

	output, err := k8s.ExecuteCommandInPod(namespace, podName, "", cmd)
	if err != nil {
		channel.Write([]byte(fmt.Sprintf("Error: %v\n", err)))
		return
	}
	channel.Write([]byte(output))
}

func handleConnection(conn net.Conn, config *cryptoSSH.ServerConfig) {
	defer conn.Close()

	sshConn, chans, reqs, err := cryptoSSH.NewServerConn(conn, config)
	if err != nil {
		log.Printf("Failed to handshake: %v", err)
		return
	}

	log.Printf("New SSH connection from %s - user: %s", sshConn.RemoteAddr(), sshConn.User())

	go cryptoSSH.DiscardRequests(reqs)

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

		go func(channel cryptoSSH.Channel, requests <-chan *cryptoSSH.Request) {
			defer channel.Close()

			req := <-requests
			if req == nil {
				return
			}

			switch req.Type {
			case "exec":
				handleExec(channel, req, sshConn.User())
			default:
				handleShell(channel, requests, sshConn.User())
			}
		}(channel, requests)
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

	hostKey, err := getOrCreateHostKey()
	if err != nil {
		log.Fatalf("Failed to get/create host key: %v", err)
	}
	config.AddHostKey(hostKey)
	ssh.StartSSHServer(config, "0.0.0.0:2222")
}
