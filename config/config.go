package config

import (
	"errors"
	"gopkg.in/yaml.v2"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"
)

// ReadCRSpecFromFile return CR config from yaml file
func ReadCRSpecFromFile(file io.Reader) (*CRSpec, error) {
	content, err := ioutil.ReadAll(file)
	if err != nil {
		log.Fatal(err)
	}
	cr := CRSpec{}
	err = yaml.Unmarshal(content, &cr)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	return &cr, nil
}

// ReadCRSpecFromEnvYaml return CR config from env yaml
func ReadCRSpecFromEnvYaml() (*CRSpec, error) {
	content := os.Getenv("YAML_CONF")
	if content == "" {
		return nil, errors.New("YAML_CONF env cannot be empty")
	}
	return ReadCRSpecFromFile(strings.NewReader(content))
}

func (cr *CRSpec) AddToConfigs(svcName, name, value string) {
	if cr.Configs[svcName] == nil {
		cr.Configs[svcName] = []NameValue{
			{
				Name:  name,
				Value: value,
			},
		}
	} else {
		//append to the config
		nv := NameValue{
			Name:  name,
			Value: value,
		}
		cr.Configs[svcName] = append(cr.Configs[svcName], nv)
	}
}

func (cr *CRSpec) AddToSecrets(svcName, name, value string) {
	if cr.Secrets[svcName] == nil {
		cr.Secrets[svcName] = []NameValue{
			{
				Name:  name,
				Value: value,
			},
		}
	} else {
		//append to the config
		nv := NameValue{
			Name:  name,
			Value: value,
		}
		cr.Secrets[svcName] = append(cr.Secrets[svcName], nv)
	}
}
