package utils

import (
	"fmt"
	"io/ioutil"

	"k8s.io/client-go/kubernetes"
	clientV1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/clientcmd"
)

const defaultNamespace = "default"

func GetSecretsClient(kubeConfigPath string) (clientV1.SecretInterface, error) {
	if config, err := clientcmd.BuildConfigFromFlags("", kubeConfigPath); err != nil {
		return nil, err
	} else if clientSet, err := kubernetes.NewForConfig(config); err != nil {
		return nil, err
	} else if namespace, err := getNamespace(kubeConfigPath); err != nil {
		return nil, err
	} else {
		return clientSet.CoreV1().Secrets(namespace), nil
	}
}

func getNamespace(kubeconfigPath string) (string, error) {
	if kubeconfigPath != "" {
		if kubeConfigContents, err := ioutil.ReadFile(kubeconfigPath); err != nil {
			return "", err
		} else if apiConfig, err := clientcmd.Load(kubeConfigContents); err != nil {
			return "", err
		} else if currentContextInfo, ok := apiConfig.Contexts[apiConfig.CurrentContext]; !ok {
			return "", fmt.Errorf("cannot extract context info for current context: %v", apiConfig.CurrentContext)
		} else {
			namespace := currentContextInfo.Namespace
			if namespace == "" {
				namespace = defaultNamespace
			}
			return namespace, nil
		}
	}
	return "", nil
}
