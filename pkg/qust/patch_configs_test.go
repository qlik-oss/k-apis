package qust

import (
	"io/ioutil"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/qlik-oss/k-apis/pkg/config"
	"gopkg.in/yaml.v2"
	"sigs.k8s.io/kustomize/api/types"
)

func TestCreateSupperConfigSelectivePatch(t *testing.T) {
	reader := setupCr(t)
	cfg, err := config.ReadCRSpecFromFile(reader)
	if err != nil {
		t.Fatalf("error reading config from file")
	}
	spMap, err := createSupperConfigSelectivePatch(cfg.Spec.Configs)
	if err != nil {
		t.Fatalf("error creating map of service selective patches")
	}
	sp := spMap["qliksense"]
	if sp.ApiVersion != "qlik.com/v1" {
		t.Fatal("ApiVersion wasn't what we expected")
	}
	if sp.Kind != "SelectivePatch" {
		t.Fatal("Kind wasn't what we expected")
	}
	if sp.Metadata["name"] != "qliksense-generated-operator-configs" {
		t.Fatal(`Metadata["name"] wasn't what we expected`)
	}
	if sp.Patches[0].Target.LabelSelector != "app=qliksense" || sp.Patches[0].Target.Kind != "SuperConfigMap" {
		t.Fatal(`patch LabelSelector or Kind wasn't what we expected`)
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
		t.Fatalf("expected %v to equal %v", scm, scm2)
	}
}

func TestProcessConfigs(t *testing.T) {
	reader := setupCr(t)
	cfg, err := config.ReadCRSpecFromFile(reader)
	if err != nil {
		t.Fatalf("error reading config from file")
	}

	td, dir := createManifestsStructure(t)

	cfg.Spec.ManifestsRoot = dir

	ProcessConfigs(cfg.Spec)
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
