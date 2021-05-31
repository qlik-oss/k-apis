package qust

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/qlik-oss/k-apis/pkg/config"
	"gopkg.in/yaml.v2"
	"sigs.k8s.io/kustomize/api/types"
	"sigs.k8s.io/kustomize/kyaml/resid"
)

func ProcessTransfomer(cr *config.CRSpec) error {
	destTransDir := filepath.Join(cr.GetManifestsRoot(), ".operator", "transformers")
	for svc, nvs := range cr.Secrets {
		for _, nv := range nvs {
			if err := writeTranasformer(destTransDir, svc, nv.Name); err != nil {
				return err
			}
		}
	}
	for svc, nvs := range cr.Configs {
		for _, nv := range nvs {
			if err := writeTranasformer(destTransDir, svc, nv.Name); err != nil {
				return err
			}
		}
	}
	// for backward qliksense-k8s compatilibity.
	return removeResourceFromKust("storage-class.yaml", filepath.Join(destTransDir, "kustomization.yaml"))
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
	patchBody := getSelectivePatchTemplateForTransformer(transformerName)
	phb, err := yaml.Marshal(patchBody)
	if err != nil {
		return types.Patch{}, err
	}
	p1 := types.Patch{
		Patch: string(phb),
	}
	p1.Target = &types.Selector{
		ResId: resid.ResId{
			Gvk: resid.Gvk{
				Kind: "SelectivePatch",
			},
		},
		LabelSelector: "app=" + appName + ",key=" + transformerName,
	}
	return p1, nil
}

func getSelectivePatchTemplateForTransformer(name string) *config.SelectivePatch {
	su := &config.SelectivePatch{
		ApiVersion: "qlik.com/v1",
		Kind:       "SelectivePatch",
		Metadata: &config.CustomMetadata{
			Name: name,
		},
		Enabled: false,
	}
	return su
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

func enabledTansformersList(baseTransDir string) ([]string, error) {
	kustFile := filepath.Join(baseTransDir, "kustomization.yaml")
	list, err := getResourcesList(kustFile)
	/*
		excludeList := []string{"storageClassName"}
		newList := make([]string, len(list))

		for _, e := range excludeList {
			for _, j := range list {
				if j != e {
					newList = append(newList, j)
				}
			}
		}
	*/
	if err != nil {
		return nil, err
	}
	result := make([]string, len(list))

	for _, l := range list {

		if isTransformerEnabled(filepath.Join(baseTransDir, l)) {
			result = append(result, l)
		}
	}
	return result, nil
}

func isTransformerEnabled(transDir string) bool {
	tfName := filepath.Base(transDir)
	kustFile := filepath.Join(transDir, "kustomization.yaml")
	list, err := getResourcesList(kustFile)
	if err != nil {
		fmt.Println("Problem getting list of resoruces from kust file" + err.Error())
		return false
	}
	for _, f := range list {
		finfo, err := os.Lstat(filepath.Join(transDir, f))
		if err != nil {
			return false
		}
		if finfo.IsDir() {
			// not expecting a director
			continue
		}
		by, err := ioutil.ReadFile(filepath.Join(transDir, f))
		if err != nil {
			fmt.Println("Cannot not read file " + err.Error())
			return false
		}

		if !strings.Contains(string(by), "kind: SelectivePatch") {
			continue
		}

		sp := &config.SelectivePatch{}
		if err := yaml.Unmarshal(by, sp); err != nil {
			fmt.Println("cannot process yaml " + err.Error())
			return false
		}
		if sp.Metadata == nil || sp.Metadata.Labels == nil {
			return false
		}
		return sp.Enabled && sp.Metadata.Labels["key"] == tfName
	}
	return false
}
