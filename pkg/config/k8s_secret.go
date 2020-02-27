package config

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ConstructK8sSecretStructure constructs a K8s Secret struct
func ConstructK8sSecretStructure(secretName string, dataMap map[string][]byte) *v1.Secret {
	return &v1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: secretName,
			// Namespace: namespace,
		},
		Type: v1.SecretTypeOpaque,
		Data: dataMap,
	}
}
