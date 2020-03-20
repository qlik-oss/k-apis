package config

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"os/exec"

	"github.com/jinzhu/copier"
	"gopkg.in/yaml.v2"
)

// ReadCRSpecFromFile return CR config from yaml file
func ReadCRSpecFromFile(file io.Reader) (*KApiCr, error) {
	content, err := ioutil.ReadAll(file)
	if err != nil {
		log.Fatal(err)
	}
	kCr := KApiCr{}
	err = yaml.Unmarshal(content, &kCr)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	return &kCr, nil
}

// ReadCRSpecFromEnvYaml return CR config from env yaml
func ReadCRSpecFromEnvYaml() (*KApiCr, error) {
	content := os.Getenv("YAML_CONF")
	if content == "" {
		return nil, errors.New("YAML_CONF env cannot be empty")
	}
	return ReadCRSpecFromFile(strings.NewReader(content))
}

func (cr *CRSpec) AddToConfigs(svcName, name, value string) {
	if cr.Configs == nil {
		cr.Configs = make(map[string]NameValues)
	}
	if cr.Configs[svcName] == nil {
		cr.Configs[svcName] = []NameValue{
			{
				Name:  name,
				Value: value,
			},
		}
		return
	}
	added := false
	for i, nn := range cr.Configs[svcName] {
		if nn.Name == name {
			cr.Configs[svcName][i] = NameValue{
				Name:  name,
				Value: value,
			}
			added = true
		}
	}
	if !added {
		nv := NameValue{
			Name:  name,
			Value: value,
		}
		cr.Configs[svcName] = append(cr.Configs[svcName], nv)
	}

}

// creates a NameValue object
func createSecretNameValue(name, value, secretName string) NameValue {
	if secretName != "" {
		return NameValue{
			Name: name,
			ValueFrom: &ValueFrom{
				SecretKeyRef: &SecretKeyRef{
					Name: secretName,
					Key:  name,
				},
			},
		}
	}

	return NameValue{
		Name:  name,
		Value: value,
	}
}

// AddToSecrets adds pieces to the secret section to the CR
// if secretName is provided value is ignored
// secretName is a kubernetes secret resource name, that already/will  exist in the cluster
func (cr *CRSpec) AddToSecrets(svcName, name, value, secretName string) {
	if cr.Secrets == nil {
		cr.Secrets = make(map[string]NameValues)
	}
	if cr.Secrets[svcName] == nil {
		cr.Secrets[svcName] = []NameValue{
			createSecretNameValue(name, value, secretName),
		}
		return
	}
	added := false
	for i, nn := range cr.Secrets[svcName] {
		if nn.Name == name {
			cr.Secrets[svcName][i] = createSecretNameValue(name, value, secretName)
			added = true
		}
	}
	if !added {
		cr.Secrets[svcName] = append(cr.Secrets[svcName], createSecretNameValue(name, value, secretName))
	}
}

// GetFromSecrets return value of the secret that exist in serets map of the spec
func (cr *CRSpec) GetFromSecrets(svcName, name string) string {
	for _, nn := range cr.Secrets[svcName] {
		if nn.Name == name {
			return getSecretValue(nn)
		}
	}
	return ""
}

// return secret value from NameValue object
func getSecretValue(nv NameValue) string {
	if nv.ValueFrom != nil {
		if va, err := readFromKubernetesSecret(nv.ValueFrom.SecretKeyRef.Name, nv.ValueFrom.SecretKeyRef.Key); err != nil {
			fmt.Println(err)
			return ""
		} else {
			return va
		}
	}
	return nv.Value
}

func (nv NameValue) GetSecretValue() string {
	return getSecretValue(nv)
}

func readFromKubernetesSecret(secName, keyName string) (string, error) {
	cmd := exec.Command("kubectl", "get", "secrets", secName, "-o", "go-template", "--template={{.data."+keyName+"}}")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		return "", err
	}
	fmt.Println(out.String())
	data, err := base64.StdEncoding.DecodeString(out.String())
	if err != nil {
		fmt.Println("error:", err)
		return "", err
	}
	return string(data), nil
}

func (in *CRSpec) DeepCopyInto(out *CRSpec) {
	copier.Copy(out, in)
}

func (in *CRSpec) DeepCopy() *CRSpec {
	if in == nil {
		return nil
	}
	out := new(CRSpec)
	in.DeepCopyInto(out)
	return out
}
func (in *KApiCr) DeepCopyInto(out *KApiCr) {
	copier.Copy(out, in)
}
func (in *KApiCr) DeepCopy() *KApiCr {
	if in == nil {
		return nil
	}
	out := new(KApiCr)
	in.DeepCopyInto(out)
	return out
}
func (cr *CRSpec) GetManifestsRoot() string {
	return cr.ManifestsRoot
}

func (cr *CRSpec) GetProfileDir() string {
	return filepath.Join("manifests", cr.Profile)
}

func (repo *Repo) GetAccessToken() (string, error) {
	if &repo.SecretName == nil {
		cmd := exec.Command("kubectl", "get", "secrets", repo.SecretName, "-o", "go-template", "--template='{{index .data accessToken}}'", "|", "base64", "-d")
		var out bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		if err != nil {
			return "", err
		}
		return out.String(), nil
	} else if &repo.AccessToken != nil {
		return repo.AccessToken, nil
	} else {
		return "", nil
	}
}
