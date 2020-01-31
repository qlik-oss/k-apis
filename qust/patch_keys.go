package qust

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/Shopify/ejson"
	"github.com/qlik-oss/k-apis/config"
	"github.com/qlik-oss/k-apis/keys"
	"gopkg.in/yaml.v2"
)

const operatorKeysBaseFolder = "keys"

type serviceT struct {
	Name       string
	PrivateKey string
	Kid        string
	JWKS       string
}

func GenerateKeys(cr *config.CRSpec, ejsonPublicKey string) error {
	serviceList, err := initServiceList(cr)
	if err != nil {
		return err
	}
	for _, service := range serviceList {
		if service.PrivateKey, service.Kid, service.JWKS, err = keys.Generate(); err != nil {
			return err
		} else if err := overrideServiceEpriviteKeyJsonFile(cr, service, ejsonPublicKey); err != nil {
			return err
		}
	}
	if err := overrideKeysEjwksJsonFile(cr, serviceList, ejsonPublicKey); err != nil {
		return err
	} else if err := overrideKeysSelectivePatchYamlFile(cr, serviceList); err != nil {
		return err
	}
	return nil
}

func initServiceList(cr *config.CRSpec) ([]*serviceT, error) {
	prePatchedSecretsDirPath := filepath.Join(cr.GetManifestsRoot(), operatorPatchBaseFolder,
		operatorKeysBaseFolder, "secrets")

	var serviceList []*serviceT
	err := filepath.Walk(prePatchedSecretsDirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			serviceList = append(serviceList, &serviceT{Name: info.Name()})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return append(serviceList[:0], serviceList[0+1:]...), nil
}

func overrideServiceEpriviteKeyJsonFile(cr *config.CRSpec, service *serviceT, ejsonPublicKey string) error {
	ePriviteKeyMap := make(map[string]string)
	ePriviteKeyMap["_public_key"] = ejsonPublicKey
	ePriviteKeyMap["private_key"] = service.PrivateKey
	ePriviteKeyMap["kid"] = service.Kid

	if err := writeToEjsonFile(ePriviteKeyMap, filepath.Join(cr.GetManifestsRoot(), operatorPatchBaseFolder,
		operatorKeysBaseFolder, "secrets", service.Name, "eprivate_key.json")); err != nil {
		return err
	}
	return nil
}

func overrideKeysEjwksJsonFile(cr *config.CRSpec, services []*serviceT, ejsonPublicKey string) error {
	eJwksMap := make(map[string]string)
	eJwksMap["_public_key"] = ejsonPublicKey
	for _, service := range services {
		eJwksMap[service.Name] = base64.StdEncoding.EncodeToString([]byte(service.JWKS))
	}

	if err := writeToEjsonFile(eJwksMap, filepath.Join(cr.GetManifestsRoot(), operatorPatchBaseFolder,
		operatorKeysBaseFolder, "configs/keys/ejwks.json")); err != nil {
		return err
	}
	return nil
}

func writeToEjsonFile(eJwksMap map[string]string, filePath string) error {
	var encryptedBuffer bytes.Buffer
	if jsonBytes, err := json.Marshal(eJwksMap); err != nil {
		return err
	} else if _, err := ejson.Encrypt(bytes.NewBuffer(jsonBytes), &encryptedBuffer); err != nil {
		return err
	} else if err := ioutil.WriteFile(filePath, encryptedBuffer.Bytes(), os.ModePerm); err != nil {
		return err
	}
	return nil
}

func overrideKeysSelectivePatchYamlFile(cr *config.CRSpec, services []*serviceT) error {
	filePath := filepath.Join(cr.GetManifestsRoot(), operatorPatchBaseFolder,
		operatorKeysBaseFolder, "configs/keys/selectivepatch.yaml")
	if selectivePatchYamlBytes, err := ioutil.ReadFile(filePath); err != nil {
		return err
	} else if transformedSelectivePatchBytes, err := updateSelectivePatchYaml(selectivePatchYamlBytes, services); err != nil {
		return err
	} else if err := ioutil.WriteFile(filePath, transformedSelectivePatchBytes, os.ModePerm); err != nil {
		return err
	}
	return nil
}

func updateSelectivePatchYaml(selectivePatchYamlBytes []byte, services []*serviceT) ([]byte, error) {
	var selectivePatchMapSlice []yaml.MapItem
	if err := yaml.Unmarshal(selectivePatchYamlBytes, &selectivePatchMapSlice); err != nil {
		return nil, err
	}
	for _, selectivePatchMapItem := range selectivePatchMapSlice {
		if selectivePatchMapItem.Key.(string) == "patches" {
			firstPatch := selectivePatchMapItem.Value.([]interface{})[0].([]yaml.MapItem)
			for i := range firstPatch {
				patchMapItem := &firstPatch[i]
				if patchMapItem.Key.(string) == "patch" {
					//unmarshal SuperConfig:
					superConfigMapString := patchMapItem.Value.(string)
					var superConfigMapSlice []yaml.MapItem
					if err := yaml.Unmarshal([]byte(superConfigMapString), &superConfigMapSlice); err != nil {
						return nil, err
					}
					//delete an existing "data" element if any:
					for j := 0; j < len(superConfigMapSlice); j++ {
						if superConfigMapSlice[j].Key.(string) == "data" {
							superConfigMapSlice = append(superConfigMapSlice[:j], superConfigMapSlice[j+1:]...)
							break
						}
					}
					//create and append a new "data" element:
					dataMapItems := yaml.MapItem{Key: "data", Value: make([]yaml.MapItem, 0)}
					for _, service := range services {
						// adding "\n" to the end of the value string to force the block scalar yaml format:
						dataMapItems.Value = append(dataMapItems.Value.([]yaml.MapItem), yaml.MapItem{
							Key:   fmt.Sprintf("qlik.api.internal-%v", service.Name),
							Value: fmt.Sprintf(`(( index (ds "data") "%v" | base64.Decode ))`, service.Name) + "\n",
						})
					}
					superConfigMapSlice = append(superConfigMapSlice, dataMapItems)
					//re-marshal SuperConfig:
					transformedSuperConfigMapBytes, err := yaml.Marshal(superConfigMapSlice)
					if err != nil {
						return nil, err
					}
					transformedSuperConfigMapString := string(transformedSuperConfigMapBytes)
					patchMapItem.Value = transformedSuperConfigMapString
					break
				}
			}
		}
	}
	transformedSelectivePatchBytes, err := yaml.Marshal(selectivePatchMapSlice)
	if err != nil {
		return nil, err
	}
	return transformedSelectivePatchBytes, nil
}
