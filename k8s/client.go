package k8s

import (
	"log"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var Clientset *kubernetes.Clientset

func InitK8sClient(kubeconfigPath string) {
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		log.Fatalf("Failed to load kubeconfig: %s", err)
	}
	Clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("Failed to create Kubernetes client: %s", err)
	}
	log.Println("Kubernetes client initialized")
}

func GetClientset() *kubernetes.Clientset {
	if Clientset == nil {
		log.Fatalf("Kubernetes clientset is not initialized. Call InitK8sClient first.")
	}
	return Clientset
}
