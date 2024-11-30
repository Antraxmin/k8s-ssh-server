package k8s

import (
	"bytes"
	"fmt"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
)

var Config *rest.Config

func ExecuteCommandInPod(namespace, podName, containerName, command string) (string, error) {
	req := Clientset.CoreV1().RESTClient().
		Post().
		Resource("pods").
		Name(podName).
		Namespace(namespace).
		SubResource("exec").
		Param("command", "/bin/sh").
		Param("command", "-c").
		Param("command", command).
		Param("stdin", "true").
		Param("stdout", "true").
		Param("stderr", "true")

	exec, err := remotecommand.NewSPDYExecutor(Config, "POST", req.URL())
	if err != nil {
		return "", fmt.Errorf("Failed to initialize executor: %w", err)
	}

	var stdout, stderr bytes.Buffer
	err = exec.Stream(remotecommand.StreamOptions{
		Stdout: &stdout,
		Stderr: &stderr,
	})
	if err != nil {
		return "", fmt.Errorf("Failed to execute command: %w", err)
	}

	return stdout.String(), nil
}

func GetPodForUser(username string) (namespace, podName string, err error) {
	switch username {
	case "user1":
		return "default", "pod-user1", nil
	case "user2":
		return "default", "pod-user2", nil
	default:
		return "", "", fmt.Errorf("No pod found for user %s", username)
	}
}
