package config

const (
	k8sSecretAPIVersion = "v1"
	k8sSecretKind       = "Secret"
	k8sSecretType       = "Opaque"
)

// ConstructK8sSecretStructure constructs a K8s Secret struct
func (k8sSecret *K8sSecret) ConstructK8sSecretStructure(secretName string, dataMap map[string]string) {
	if k8sSecret != nil {
		k8sSecret.APIVersion = k8sSecretAPIVersion
		k8sSecret.Kind = k8sSecretKind
		k8sSecret.Name = secretName
		k8sSecret.Type = k8sSecretType
		k8sSecret.Data = dataMap
	}
}
