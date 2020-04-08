package qust

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"

	"gopkg.in/yaml.v2"
	machine_yaml "k8s.io/apimachinery/pkg/util/yaml"
)

func K8sToYaml(k8sObj interface{}) ([]byte, error) {
	k8sSecretYamlMap := map[string]interface{}{}
	if k8sSecretYamlBytes, err := yaml.Marshal(k8sObj); err != nil {
		return nil, err
	} else if err := yaml.Unmarshal(k8sSecretYamlBytes, &k8sSecretYamlMap); err != nil {
		return nil, err
	} else {
		delete(k8sSecretYamlMap["metadata"].(map[string]interface{}), "creationTimestamp")
		return yaml.Marshal(k8sSecretYamlMap)
	}
}

// WriteToFile (content, targetFile) writes content into specified file
func WriteToFile(content interface{}, targetFile string) error {
	if content == nil || targetFile == "" {
		return nil
	}

	x, err := K8sToYaml(content)
	if err != nil {
		err = fmt.Errorf("An error occurred during marshalling CR: %v", err)
		log.Println(err)
		return err
	}
	// Writing content
	err = ioutil.WriteFile(targetFile, x, 0644)
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}

// ReadFromFile (content, targetFile) reads content from specified sourcefile
func ReadFromFile(content interface{}, sourceFile string) error {
	if content == nil || sourceFile == "" {
		return nil
	}
	file, e := os.Open(sourceFile)
	if e != nil {
		return e
	}
	return ReadFromStream(content, file)
}

// ReadFromStream reads from input stream and creat yaml struct of type content
func ReadFromStream(content interface{}, reader io.Reader) error {
	contents, err := ioutil.ReadAll(reader)
	if err != nil {
		err = fmt.Errorf("There was an error reading from reader: %v", err)
		return err
	}
	// reading k8s style object
	// https://stackoverflow.com/questions/44306554/how-to-deserialize-kubernetes-yaml-file
	dec := machine_yaml.NewYAMLOrJSONDecoder(bytes.NewReader(contents), 10000)
	return dec.Decode(content)
}
