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
