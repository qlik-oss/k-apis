package config

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	Git              *Repo                 `json:"git,omitempty" yaml:"git,omitempty"`
	OpsRunner        *OpsRunner            `json:"opsRunner,omitempty" yaml:"opsRunner,omitempty"`
	TlsCertHost      string                `json:"tlsCertHost,omitempty" yaml:"tlsCertHost,omitempty"`
	TlsCertOrg       string                `json:"tlsCertOrg,omitempty" yaml:"tlsCertOrg,omitempty"`
}

type KApiCr struct {
	metav1.TypeMeta   `json:",inline" yaml:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	Spec              *CRSpec `json:"spec" yaml:"spec"`
}

type SelectivePatch struct {
	ApiVersion string          `yaml:"apiVersion"`
	Kind       string          `yaml:"kind"`
	Metadata   *CustomMetadata `yaml:"metadata"`
	Enabled    bool            `yaml:"enabled"`
	Default    bool            `yaml:"default,omitempty"`
	Patches    []types.Patch   `yaml:"patches,omitempty"`
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
	Name      string     `yaml:"name,omitempty" json:"name,omitempty"`
	Value     string     `yaml:"value,omitempty" json:"value,omitempty"`
	ValueFrom *ValueFrom `yaml:"valueFrom,omitempty" json:"valueFrom,omitempty"`
}

type ValueFrom struct {
	SecretKeyRef *SecretKeyRef `yaml:"secretKeyRef" json:"secretKeyRef"`
}

type SecretKeyRef struct {
	Name string `yaml:"name,omitempty" json:"name,omitempty"`
	Key  string `yaml:"key,omitempty" json:"key,omitempty"`
}

type Repo struct {
	Repository  string `json:"repository,omitempty" yaml:"repository,omitempty"`
	UserName    string `json:"userName,omitempty" yaml:"userName,omitempty"`
	Password    string `json:"password,omitempty" yaml:"password,omitempty"`
	AccessToken string `json:"accessToken,omitempty" yaml:"accessToken,omitempty"`
	SecretName  string `json:"secretName,omitempty" yaml:"secretName,omitempty"`
}

type OpsRunner struct {
	Enabled     string `json:"enabled,omitempty" yaml:"enabled,omitempty"`
	Schedule    string `json:"schedule,omitempty" yaml:"schedule,omitempty"`
	WatchBranch string `json:"watchBranch,omitempty" yaml:"watchBranch,omitempty"`
	Image       string `json:"image,omitempty" yaml:"image,omitempty"`
}

type CustomMetadata struct {
	Name        string            `json:"name,omitempty" yaml:"name,omitempty"`
	Labels      map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty" yaml:"annotations,omitempty"`
}
