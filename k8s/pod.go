package k8s

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func CreateUserPod(username string) (string, error) {
	clientset := GetClientset()
	podName := fmt.Sprintf("pod-%s", strings.ToLower(username))

	pod, err := clientset.CoreV1().Pods("default").Get(context.TODO(), podName, metav1.GetOptions{})
	if err == nil && pod.Status.Phase == corev1.PodRunning {
		return podName, nil
	}

	pod = &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: podName,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "nginx",
					Image: "nginx",
					Ports: []corev1.ContainerPort{
						{ContainerPort: 80},
					},
				},
			},
		},
	}

	_, err = clientset.CoreV1().Pods("default").Create(context.TODO(), pod, metav1.CreateOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to create pod: %v", err)
	}

	return podName, nil
}
