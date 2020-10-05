package qust

import (
	"io/ioutil"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/Shopify/ejson"

	"github.com/qlik-oss/k-apis/pkg/config"
	"gopkg.in/yaml.v2"
	"sigs.k8s.io/kustomize/api/types"
)

func TestCreateSupperSecretSelectivePatch(t *testing.T) {
	reader := setupCr(t)
	cfg, err := config.ReadCRSpecFromFile(reader)
	if err != nil {
		t.Fatal("error reading config from file")
	}
	spMap, err := createSupperSecretSelectivePatch(cfg.Spec.Secrets)
	if err != nil {
		t.Fatal("error creating map of service selective patches")
	}
	sp := spMap["qliksense"]
	if sp.ApiVersion != "qlik.com/v1" {
		t.Fatal("ApiVersion wasn't what we expected")
	}
	if sp.Kind != "SelectivePatch" {
		t.Fatal("Kind wasn't what we expected")
	}
	if sp.Metadata.Name != "qliksense-generated-operator-secrets" {
		t.Fatal(`Metadata["name"] wasn't what we expected`)
	}
	if sp.Patches[0].Target.LabelSelector != "app=qliksense" || sp.Patches[0].Target.Kind != "SuperSecret" {
		t.Fatal(`patch LabelSelector or Kind wasn't what we expected`)
	}
	ss := &config.SupperSecret{
		ApiVersion: "qlik.com/v1",
		Kind:       "SuperSecret",
		Metadata: map[string]string{
			"name": "qliksense-secrets",
		},
		Data: map[string]string{
			"mongodbUri": `(( index (ds "data") "mongodbUri" ))`,
		},
	}
	ss2 := &config.SupperSecret{}
	if err := yaml.Unmarshal([]byte(sp.Patches[0].Patch), ss2); err != nil {
		t.Fail()
		t.Log(err)
	}
	if !reflect.DeepEqual(ss, ss2) {
		t.Fail()
		t.Log("expected selectivePatch: ", ss)
		t.Log("Actual SelectivePatch: ", ss2)
	}
}

func TestProcessCrSecrets(t *testing.T) {
	reader := setupCr(t)
	cfg, err := config.ReadCRSpecFromFile(reader)
	if err != nil {
		t.Fatalf("error reading config from file")
	}

	td, dir := createManifestsStructure(t)

	cfg.Spec.ManifestsRoot = dir

	ejsonPublicKey, _, err := ejson.GenerateKeypair()
	if err != nil {
		t.Fatalf("error generating ejson keys")
	}

	err = ProcessSecrets(cfg.Spec, ejsonPublicKey)
	if err != nil {
		t.Fatalf("unexpected error processing secrets")
	}

	content, _ := ioutil.ReadFile(filepath.Join(dir, ".operator", "secrets", "qliksense", "selectivepatch.yaml"))

	sp := getSuperSecretSPTemplate("qliksense")
	scm := getSuperSecretTemplate("qliksense")
	scm.Data = map[string]string{
		"mongodbUri": `(( index (ds "data") "mongodbUri" ))`,
	}
	phb, _ := yaml.Marshal(scm)
	sp.Patches = []types.Patch{
		{
			Patch:  strings.Replace(string(phb), ": |", ": |-", -1),
			Target: getSelector("SuperSecret", "qliksense"),
		},
	}
	spOut := &config.SelectivePatch{}
	yaml.Unmarshal(content, spOut)
	if !reflect.DeepEqual(sp, spOut) {
		t.Fail()
	}

	td()
}
