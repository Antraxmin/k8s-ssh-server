package ssh

import (
	"fmt"
	"log"
	"net"

	"k8s-ssh-server/k8s"

	"golang.org/x/crypto/ssh"
)

func handleConnection(conn net.Conn, config *ssh.ServerConfig) {
	sshConn, channels, requests, err := ssh.NewServerConn(conn, config)
	if err != nil {
		log.Printf("Failed to handshake: %s", err)
		return
	}
	defer sshConn.Close()
	log.Printf("New SSH connection from %s (%s)", sshConn.RemoteAddr(), sshConn.ClientVersion())

	go ssh.DiscardRequests(requests)

	for newChannel := range channels {
		if newChannel.ChannelType() != "session" {
			newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
			continue
		}
		channel, requests, err := newChannel.Accept()
		if err != nil {
			log.Printf("Failed to accept channel: %s", err)
			continue
		}
		go handleChannel(channel, requests, sshConn.User())
	}
}

func handleChannel(channel ssh.Channel, requests <-chan *ssh.Request, username string) {
	defer channel.Close()

	for req := range requests {
		switch req.Type {
		case "exec":
			cmd := string(req.Payload[4:])
			namespace, podName, err := k8s.GetPodForUser(username)
			if err != nil {
				channel.Write([]byte(fmt.Sprintf("Error: %s\n", err.Error())))
				req.Reply(false, nil)
				continue
			}
			output, err := k8s.ExecuteCommandInPod(namespace, podName, "", cmd)
			if err != nil {
				channel.Write([]byte(fmt.Sprintf("Failed to execute command: %s\n", err.Error())))
				req.Reply(false, nil)
				continue
			}
			channel.Write([]byte(output))
			req.Reply(true, nil)
		default:
			req.Reply(false, nil)
		}
	}
}
