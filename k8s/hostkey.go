package k8s

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"

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
	privateKeyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privateKeyBytes,
	})

	secret, err := Clientset.CoreV1().Secrets("default").Get(context.TODO(), "ssh-host-key", metav1.GetOptions{})
	if err != nil {
		secret = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ssh-host-key",
			},
			Data: map[string][]byte{
				"key": privateKeyPEM,
			},
		}
		_, err = Clientset.CoreV1().Secrets("default").Create(context.TODO(), secret, metav1.CreateOptions{})
	} else {
		secret.Data = map[string][]byte{
			"key": privateKeyPEM,
		}
		_, err = Clientset.CoreV1().Secrets("default").Update(context.TODO(), secret, metav1.UpdateOptions{})
	}
	return err
}
