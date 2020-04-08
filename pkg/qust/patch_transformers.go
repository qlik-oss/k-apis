package qust

import (
	"io/ioutil"
	"path/filepath"

	"github.com/qlik-oss/k-apis/pkg/config"
	"gopkg.in/yaml.v2"
	"sigs.k8s.io/kustomize/api/types"
)

func ProcessTransfomer(cr *config.CRSpec) error {
	transformersDir := filepath.Join(cr.GetManifestsRoot(), "manifests", "base", "transformers")
	destTransDir := filepath.Join(cr.GetManifestsRoot(), ".operator", "transformers")

	list, err := disabledTansformersList(transformersDir)
	if err != nil {
		return err
	}
	for svc, nvs := range cr.Secrets {
		for _, nv := range nvs {
			if contains(list, nv.Name) {
				if err := writeTranasformer(destTransDir, svc, nv.Name); err != nil {
					return err
				}
			}
		}
	}
	for svc, nvs := range cr.Configs {
		for _, nv := range nvs {
			if contains(list, nv.Name) {
				if err := writeTranasformer(destTransDir, svc, nv.Name); err != nil {
					return err
				}
			}

		}
	}
	return nil
}

func writeTranasformer(transDir, appName, transformerName string) error {
	appFileName := appName + ".yaml"
	appFilePath := filepath.Join(transDir, appFileName)
	kustFile := filepath.Join(transDir, "kustomization.yaml")
	sp, err := loadExistingOrCreateEmptySelectivePatch(appName, appName+"-operator-generated", transDir)
	if err != nil {
		return err
	}
	p, err := createSelectivePatchObjectForTransformer(transformerName, appName)
	if err != nil {
		return err
	}
	sp.Patches = append(sp.Patches, p)
	if spBytes, err := yaml.Marshal(sp); err != nil {
		return err
	} else if err := ioutil.WriteFile(appFilePath, spBytes, FILE_PERMISION); err != nil {
		return err
	} else {
		return addResourceToKustomization(appFileName, kustFile)
	}
}

/**
The geenrated patch for the transformer caCertificates, service audit will look like this

- target:
   kind: SelectivePatch
   #labelSelector: app=audit
 patch: |-
   apiVersion: qlik.com/v1
   kind: SelectivePatch
   metadata:
     name: caCertificates
   enabled: true
*/

func createSelectivePatchObjectForTransformer(transformerName, appName string) (types.Patch, error) {
	//patchName := "transformer-"
	//sp := getSelectivePatchTemplate(patchName)
	patchBody := getSelectivePatchTemplate(transformerName)
	phb, err := yaml.Marshal(patchBody)
	if err != nil {
		return types.Patch{}, err
	}
	p1 := types.Patch{
		Patch: string(phb),
	}
	if appName == "qliksense" {
		p1.Target = getSelector("SelectivePatch", "")
	} else {
		p1.Target = getSelector("SelectivePatch", appName)
	}
	//sp.Patches = []types.Patch{p1}
	return p1, nil
}

/**
loadExistingOrCreateEmptySelectivePatch create a selective patch with the name spName, if not already exist for the app
generated yaml look like this

apiVersion: qlik.com/v1
kind: SelectivePatch
metadata:
  name: spName
enabled: true
*/
func loadExistingOrCreateEmptySelectivePatch(appName, spName, kustDirectory string) (*config.SelectivePatch, error) {
	sp := &config.SelectivePatch{}

	kustFile := filepath.Join(kustDirectory, "kustomization.yaml")

	list, err := getResourcesList(kustFile)
	if err != nil {
		return nil, err
	}

	appFileName := appName + ".yaml"
	if contains(list, appFileName) {
		appFilePath := filepath.Join(kustDirectory, appFileName)
		if content, err := ioutil.ReadFile(appFilePath); err != nil {
			return nil, err
		} else if err := yaml.Unmarshal(content, sp); err != nil {
			return nil, err
		} else {
			return sp, nil
		}
	}
	return getSelectivePatchTemplate(spName), nil
}

func disabledTansformersList(baseTransDir string) ([]string, error) {
	kustFile := filepath.Join(baseTransDir, "kustomization.yaml")
	list, err := getResourcesList(kustFile)

	if err != nil {
		return nil, err
	}
	result := make([]string, len(list))

	for _, l := range list {
		sp := &config.SelectivePatch{}
		tKFile := filepath.Join(baseTransDir, l, "selectivepatch.yaml")
		if b, e := ioutil.ReadFile(tKFile); e != nil {
			// file not found
			continue
		} else if e := yaml.Unmarshal(b, sp); e != nil {
			return nil, e
		} else if !sp.Enabled {
			result = append(result, l)
		}
	}
	return result, nil
}
