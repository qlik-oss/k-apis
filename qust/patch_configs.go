package qust

import (
	"github.com/qlik-oss/k-apis/config"
	"gopkg.in/yaml.v2"
	"log"
	"os"
	"path/filepath"
	"sigs.k8s.io/kustomize/api/types"
)

func ProcessConfigs(cr *config.CRSpec) {
	baseConfigDir := filepath.Join(cr.ManifestsRoot, operatorPatchBaseFolder, "configs")
	if _, err := os.Stat(baseConfigDir); os.IsNotExist(err) {
		log.Panic(baseConfigDir + " does not exist ")
	}
	pm := createSupperConfigSelectivePatch(cr.Configs)
	for svc, sps := range pm {
		fpath := filepath.Join(baseConfigDir, svc+".yaml")
		fileHand, _ := os.Create(fpath)
		YamlToWriter(fileHand, sps)
		err := addResourceToKustomization(svc+".yaml", filepath.Join(baseConfigDir, "kustomization.yaml"))
		if err != nil {
			log.Println("Cannot process configs", err)
		}
	}
}

// create a selectivepatch map for each service for a dataKey
func createSupperConfigSelectivePatch(confg map[string][]config.NameValue) map[string]*config.SelectivePatch {
	spMap := make(map[string]*config.SelectivePatch)
	for svc, data := range confg {
		sp := getSuperConfigSPTemplate(svc)
		for _, conf := range data {
			p := getConfigMapPatchBody(conf.Name, svc, conf.Value)
			sp.Patches = []types.Patch{p}
			mergeSelectivePatches(sp, spMap[svc])
			spMap[svc] = sp
		}
	}
	return spMap
}

// create a patch section to be added to the selective patch
func getConfigMapPatchBody(dataKey, svc, value string) types.Patch {
	ph := getSuperConfigMapTemplate(svc)
	ph.Data = map[string]string{
		dataKey: value,
	}
	// ph := `
	// 	apiVersion: qlik.com/v1
	// 	kind: SuperConfigMap
	// 	metadata:
	// 		name: ` + svc + `-configs
	// 	data:
	// 		` + dataKey + `: ` + value

	// target:
	//   kind: SuperConfigMap
	//   labelSelector: "app=" + svc,
	phb, _ := yaml.Marshal(ph)
	p1 := types.Patch{
		Patch:  string(phb),
		Target: getSelector("SuperConfigMap", svc),
	}
	return p1
}

// a SelectivePatch object with service name in it
func getSuperConfigSPTemplate(svc string) *config.SelectivePatch {
	su := &config.SelectivePatch{
		ApiVersion: "qlik.com/v1",
		Kind:       "SelectivePatch",
		Metadata: map[string]string{
			"name": svc + "-operator-configs",
		},
		Enabled: true,
	}
	return su
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
