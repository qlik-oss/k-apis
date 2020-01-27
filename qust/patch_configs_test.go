package qust

import (
	"github.com/qlik-oss/k-apis/config"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"path/filepath"
	"reflect"
	"sigs.k8s.io/kustomize/api/types"
	"testing"
)

func TestCreateSupperConfigSelectivePatch(t *testing.T) {
	reader := setupCr(t)
	cfg, err := config.ReadCRConfigFromFile(reader)
	if err != nil {
		t.Fatalf("error reading config from file")
	}
	spMap := createSupperConfigSelectivePatch(cfg.Configs)
	sp := spMap["qliksense"]
	if sp.ApiVersion != "qlik.com/v1" {
		t.Fail()
	}
	if sp.Kind != "SelectivePatch" {
		t.Fail()
	}
	if sp.Metadata["name"] != "qliksense-operator-configs" {
		t.Fail()
	}
	if sp.Patches[0].Target.LabelSelector != "app=qliksense" || sp.Patches[0].Target.Kind != "SuperConfigMap" {
		t.Fail()
	}
	scm := &config.SupperConfigMap{
		ApiVersion: "qlik.com/v1",
		Kind:       "SuperConfigMap",
		Metadata: map[string]string{
			"name": "qliksense-configs",
		},
		Data: map[string]string{
			"acceptEULA": "yes",
		},
	}
	scm2 := &config.SupperConfigMap{}
	yaml.Unmarshal([]byte(sp.Patches[0].Patch), scm2)
	if !reflect.DeepEqual(scm, scm2) {
		t.Fail()
	}
}

func TestProcessCrConfigs(t *testing.T) {
	reader := setupCr(t)
	cfg, err := config.ReadCRConfigFromFile(reader)
	if err != nil {
		t.Fatalf("error reading config from file")
	}

	td, dir := createManifestsStructure(t)

	cfg.ManifestsRoot = dir
	ProcessCrConfigs(cfg)
	content, _ := ioutil.ReadFile(filepath.Join(dir, ".operator", "configs", "qliksense.yaml"))

	sp := getSuperConfigSPTemplate("qliksense")
	scm := getSuperConfigMapTemplate("qliksense")
	scm.Data = map[string]string{
		"acceptEULA": "yes",
	}
	phb, _ := yaml.Marshal(scm)
	sp.Patches = []types.Patch{
		types.Patch{
			Patch:  string(phb),
			Target: getSelector("SuperConfigMap", "qliksense"),
		},
	}
	spOut := &config.SelectivePatch{}
	yaml.Unmarshal(content, spOut)
	if !reflect.DeepEqual(sp, spOut) {
		t.Fail()
	}

	td()
}
