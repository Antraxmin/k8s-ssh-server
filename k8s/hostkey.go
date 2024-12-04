package k8s

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GetHostKey() ([]byte, error) {
	secret, err := Clientset.CoreV1().Secrets("default").Get(context.TODO(), "ssh-host-key", metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return secret.Data["key"], nil
}

func SaveHostKey(privateKey *rsa.PrivateKey) error {
	log.Printf("Attempting to save host key...")

	privateKeyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privateKeyBytes,
	})

	log.Printf("Checking if secret exists...")
	secret, err := Clientset.CoreV1().Secrets("default").Get(context.TODO(), "ssh-host-key", metav1.GetOptions{})
	if err != nil {
		log.Printf("Secret does not exist, creating new one: %v", err)
		secret = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "ssh-host-key",
				Namespace: "default",
			},
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{
				"key": privateKeyPEM,
			},
		}
		log.Printf("Attempting to create secret...")
		_, err = Clientset.CoreV1().Secrets("default").Create(context.TODO(), secret, metav1.CreateOptions{})
		if err != nil {
			log.Printf("Failed to create secret: %v", err)
			return fmt.Errorf("failed to create secret: %v", err)
		}
	} else {
		log.Printf("Secret exists, updating...")
		secret.Data = map[string][]byte{
			"key": privateKeyPEM,
		}
		_, err = Clientset.CoreV1().Secrets("default").Update(context.TODO(), secret, metav1.UpdateOptions{})
		if err != nil {
			log.Printf("Failed to update secret: %v", err)
			return fmt.Errorf("failed to update secret: %v", err)
		}
	}

	log.Printf("Successfully saved host key")
	return nil
}
