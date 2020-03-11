package qust

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/qlik-oss/k-apis/pkg/config"
	"gopkg.in/yaml.v2"
	"sigs.k8s.io/kustomize/api/types"
)

func ProcessConfigs(cr *config.CRSpec) error {
	baseConfigDir := filepath.Join(cr.GetManifestsRoot(), operatorPatchBaseFolder, "configs")
	if _, err := os.Stat(baseConfigDir); os.IsNotExist(err) {
		return fmt.Errorf("%v does not exist", baseConfigDir)
	} else if pm, err := createSupperConfigSelectivePatch(cr.Configs); err != nil {
		return errors.Wrap(err, "error creating the selective patches map")
	} else {
		for svc, sps := range pm {
			if spsBytes, err := yaml.Marshal(sps); err != nil {
				return errors.Wrap(err, "error marshalling selective patch")
			} else if err := ioutil.WriteFile(filepath.Join(baseConfigDir, fmt.Sprintf("%v.yaml", svc)), spsBytes, os.ModePerm); err != nil {
				return errors.Wrap(err, "error writing out the selective patch")
			} else if err := addResourceToKustomization(fmt.Sprintf("%v.yaml", svc), filepath.Join(baseConfigDir, "kustomization.yaml")); err != nil {
				return errors.Wrapf(err, "error adding %v to the kustomization.yaml", fmt.Sprintf("%v.yaml", svc))
			}
		}
	}
	return nil
}

// create a selectivepatch map for each service for a dataKey
func createSupperConfigSelectivePatch(confg map[string]config.NameValues) (map[string]*config.SelectivePatch, error) {
	spMap := make(map[string]*config.SelectivePatch)
	for svc, data := range confg {
		spMap[svc] = getSuperConfigSPTemplate(svc)
		for _, conf := range data {
			sp := getSuperConfigSPTemplate(svc)
			sp.Patches = []types.Patch{getConfigMapPatchBody(conf.Name, svc, conf.Value)}
			if _, err := mergeSelectivePatches(spMap[svc], sp); err != nil {
				return nil, err
			}
		}
	}
	return spMap, nil
}

// create a patch section to be added to the selective patch
func getConfigMapPatchBody(dataKey, svc, value string) types.Patch {
	ph := getSuperConfigMapTemplate(svc)
	ph.Data = map[string]string{
		dataKey: value,
	}
	phb, _ := yaml.Marshal(ph)
	p1 := types.Patch{
		Patch:  string(phb),
		Target: getSelector("SuperConfigMap", svc),
	}
	return p1
}

// a SelectivePatch object with service name in it
func getSuperConfigSPTemplate(svc string) *config.SelectivePatch {
	return getSelectivePatchTemplate(svc + "-operator-configs")
}

func getSuperConfigMapTemplate(svc string) *config.SupperConfigMap {
	return &config.SupperConfigMap{
		ApiVersion: "qlik.com/v1",
		Kind:       "SuperConfigMap",
		Metadata: map[string]string{
			"name": svc + "-configs",
		},
	}
}
