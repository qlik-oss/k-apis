package config

import (
	"sigs.k8s.io/kustomize/api/types"
)

// CRSpec defines the configuration for the whole manifests
// It is expecting in the manifestsRoot folder two subfolders .operator and .configuration exist
// operator will add patch into .operator folder
// customer will add patch into .configuration folder
type CRSpec struct {
	// relative to manifestsRoot folder, ex. ./manifests/base
	Profile          string                `json:"profile" yaml:"profile"`
	Secrets          map[string]NameValues `json:"secrets,omitempty" yaml:"secrets,omitempty"`
	Configs          map[string]NameValues `json:"configs,omitempty" yaml:"configs,omitempty"`
	ManifestsRoot    string                `json:"manifestsRoot,omitempty" yaml:"manifestsRoot,omitempty"`
	RotateKeys       string                `json:"rotateKeys,omitempty" yaml:"rotateKeys,omitempty"`
	StorageClassName string                `json:"storageClassName,omitempty" yaml:"storageClassName,omitempty"`
	NameSpace        string                `json:"namespace,omitempty" yaml:"namespace,omitempty"`
	Git              Repo                  `json:"git,omitempty" yaml:"git,omitempty"`
	ReleaseName      string                `json:"releaseName,omitempty" yaml:"releaseName,omitempty"`
}

type SelectivePatch struct {
	ApiVersion string            `yaml:"apiVersion"`
	Kind       string            `yaml:"kind"`
	Metadata   map[string]string `yaml:"metadata"`
	Enabled    bool              `yaml:"enabled,omitempty"`
	Patches    []types.Patch     `yaml:"patches,omitempty"`
}

type SupperConfigMap struct {
	ApiVersion string            `yaml:"apiVersion"`
	Kind       string            `yaml:"kind"`
	Metadata   map[string]string `yaml:"metadata,omitempty"`
	Data       map[string]string `yaml:"data,omitempty"`
}
type SupperSecret struct {
	ApiVersion string            `yaml:"apiVersion"`
	Kind       string            `yaml:"kind"`
	Metadata   map[string]string `yaml:"metadata,omitempty"`
	Data       map[string]string `yaml:"data,omitempty"`
	StringData map[string]string `yaml:"stringData,omitempty"`
}

// operator-sdk needs named type
type NameValues []NameValue

type NameValue struct {
	Name         string `yaml:"name" json:"name"`
	Value        string `yaml:"value,omitempty" json:"value,omitempty"`
	ValueFromKey string `yaml:"valueFromKey,omitempty" json:"valueFromKey,omitempty"`
}

type Repo struct {
	Repository  string `json:"repository"`
	UserName    string `json:"userName,omitempty" yaml:"userName,omitempty"`
	Password    string `json:"password,omitempty" yaml:"password,omitempty"`
	AccessToken string `json:"accessToken,omitempty" yaml:"accessToken,omitempty"`
	SecretName  string `json:"secretName,omitempty" yaml:"secretName,omitempty"`
}
