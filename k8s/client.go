package k8s

import (
	"log"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var Clientset *kubernetes.Clientset

func InitK8sClient(kubeconfigPath string) {
	var err error
	Config, err = rest.InClusterConfig()
	if err != nil {
		Config, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
		if err != nil {
			log.Fatalf("Failed to create config: %v", err)
		}
	}

	Clientset, err = kubernetes.NewForConfig(Config)
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
