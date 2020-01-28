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

// ReadCRConfigFromFile return CR config from yaml file
func ReadCRConfigFromFile(file io.Reader) (*CRConfig, error) {
	content, err := ioutil.ReadAll(file)
	if err != nil {
		log.Fatal(err)
	}
	cr := CRConfig{}
	err = yaml.Unmarshal(content, &cr)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	return &cr, nil
}

// ReadCRConfigFromEnvYaml return CR config from env yaml
func ReadCRConfigFromEnvYaml() (*CRConfig, error) {
	content := os.Getenv("YAML_CONF")
	if content == "" {
		return nil, errors.New("YAML_CONF env cannot be empty")
	}
	return ReadCRConfigFromFile(strings.NewReader(content))
}

func (cr *CRConfig) AddToConfigs(svcName, name, value string) {
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

func (cr *CRConfig) AddToSecrets(svcName, name, value string) {
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
