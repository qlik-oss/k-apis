package config

import (
	"fmt"

	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	clientV1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/clientcmd"

	kubeApiErrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	defaultNamespace = "qlik-default"
	secretAPIVersion = "v1"
)

func getSecretsClient(kubeconfigPath string, namespaceName string) (clientV1.SecretInterface, error) {
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return nil, err
	}

	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	secretsClient := clientSet.CoreV1().Secrets(namespaceName)
	return secretsClient, nil
}

// GetK8sSecretData retrieves a Kubernetes Secret data
func GetK8sSecretData(namespace, kubeconfigPath, secretName string) (map[string][]byte, error) {
	if namespace == "" {
		namespace = defaultNamespace
	}

	if secretName == "" {
		return nil, fmt.Errorf("secret name is empty")
	}

	secretsClient, err := getSecretsClient(kubeconfigPath, namespace)
	if err != nil {
		return nil, err
	}

	secret, err := secretsClient.Get(secretName, metaV1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return secret.Data, nil
}

// CreateK8sSecret creates a Kuberenetes Secret
func CreateK8sSecret(namespace, kubeconfigPath, secretName string, dataMap map[string][]byte) error {
	if namespace == "" {
		namespace = defaultNamespace
	}

	if secretName == "" {
		return fmt.Errorf("secret name is empty")
	}

	secretsClient, err := getSecretsClient(kubeconfigPath, namespace)
	if err != nil {
		return err
	}

	secret, err := secretsClient.Get(secretName, metaV1.GetOptions{})
	if err != nil && kubeApiErrors.IsNotFound(err) {
		//doesn't exist, create:
		if _, err := secretsClient.Create(&v1.Secret{
			TypeMeta: metaV1.TypeMeta{
				APIVersion: secretAPIVersion,
				Kind:       "Secret",
			},
			ObjectMeta: metaV1.ObjectMeta{
				Name: secretName,
			},
			Type: v1.SecretTypeOpaque,
			Data: dataMap,
		}); err != nil {
			return err
		}
		// done creating the secret
		return nil
	} else if err == nil {
		//exists, update (overwrite)
		secret.Data = dataMap
		_, err = secretsClient.Update(secret)
		if err != nil {
			return err
		}
	}
	return err
}
