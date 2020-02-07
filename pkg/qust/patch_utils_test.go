package qust

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v2"
	"sigs.k8s.io/kustomize/api/types"
)

const tempPermissionCode os.FileMode = 0777

func setup() (func(), string) {
	dir, _ := ioutil.TempDir("", "testing_path")
	kustFile := `
kind: Kustomization
apiversion: kustomize.config.k8s.io/v1beta1
transformers:
- test-transformer.yaml
patches:
- path: test-patch.yaml
resources:
- mongodb-secret.yaml`
	kustf := filepath.Join(dir, "kustomization.yaml")
	ioutil.WriteFile(kustf, []byte(kustFile), tempPermissionCode)
	tearDown := func() {
		os.RemoveAll(dir)
	}
	return tearDown, dir
}

func setupCr(t *testing.T) io.Reader {
	t.Parallel()
	sampleConfig := `
profile: base
manifestsRoot: "./manifests"
storageClassName: "efs"
configs:
  qliksense:
  - name: acceptEULA
    value: "yes"
secrets:
  qliksense:
  - name: mongoDbUri
    value: mongo://mongo:3307`
	os.Setenv("YAML_CONF", sampleConfig)
	return strings.NewReader(sampleConfig)
}

// it create manifest structure and return a td function to delete them latter
// and the path to the root directory of manifests
func createManifestsStructure(t *testing.T) (func(), string) {
	/*
		manifestsRoot
		|--.operator
		   |--configs
					|--kustomization.yaml
			 |--secrets
					|--kustomization.yaml
			 |--transformers
					|--kustomization.yaml
					|--storage-class.yaml
	*/
	dir, _ := ioutil.TempDir("", "test_manifests")
	oprCnfDir := filepath.Join(dir, ".operator", "configs")
	oprSecDir := filepath.Join(dir, ".operator", "secrets")
	oprTansDir := filepath.Join(dir, ".operator", "transformers")
	os.MkdirAll(oprCnfDir, tempPermissionCode)
	os.MkdirAll(oprSecDir, tempPermissionCode)
	os.MkdirAll(oprTansDir, tempPermissionCode)

	k := `
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:`

	err := ioutil.WriteFile(filepath.Join(oprCnfDir, "kustomization.yaml"), []byte(k), tempPermissionCode)
	if err != nil {
		t.Log(err)
		os.Exit(1)
	}
	ioutil.WriteFile(filepath.Join(oprSecDir, "kustomization.yaml"), []byte(k), tempPermissionCode)
	stk := `
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
- storage-class.yaml
`
	scf := `
apiVersion: qlik.com/v1
kind: SelectivePatch
enabled: true
patches:
- target:
		name: storageClassName
		labelSelector: app=engine
	patch: |-
		- op: replace
			path: /enabled
			value: false
- target:
		name: storageClassName
		labelSelector: app=qix-datafiles
	patch: |-
		- op: replace
			path: /enabled
			value: false`
	err = ioutil.WriteFile(filepath.Join(oprTansDir, "kustomization.yaml"), []byte(stk), tempPermissionCode)
	if err != nil {
		t.Log(err)
		os.Exit(1)
	}
	err = ioutil.WriteFile(filepath.Join(oprTansDir, "storage-class-template.yaml"), []byte(scf), tempPermissionCode)
	if err != nil {
		t.Log(err)
		os.Exit(1)
	}

	tearDown := func() {
		os.RemoveAll(dir)
	}
	return tearDown, dir
}

func TestAddResourceToKustomization(t *testing.T) {
	td, dir := setup()
	kustFile := filepath.Join(dir, "kustomization.yaml")
	addResourceToKustomization("test-file.yaml", kustFile)
	kust := &types.Kustomization{}
	content, err := ioutil.ReadFile(kustFile)
	if err != nil {
		t.FailNow()
	}
	yaml.Unmarshal(content, kust)
	if kust.Resources[1] != "test-file.yaml" {
		t.Fail()
	}
	addResourceToKustomization("test/test-file.yaml", kustFile)
	kust = &types.Kustomization{}
	content, err = ioutil.ReadFile(kustFile)
	if err != nil {
		t.FailNow()
	}
	yaml.Unmarshal(content, kust)
	if kust.Resources[2] != "test/test-file.yaml" {
		t.Fail()
	}
	td()
}

// func TestCreateSupperConfigSelectivePatch(t *testing.T) {
// 	reader := setupCr(t)
// 	cfg, err := config.ReadCRSpecFromFile(reader)
// 	if err != nil {
// 		t.Fatalf("error reading config from file")
// 	}
// 	spMap := createSupperConfigSelectivePatch(cfg.Configs)
// 	sp := spMap["qliksense"]
// 	if sp.ApiVersion != "qlik.com/v1" {
// 		t.Fail()
// 	}
// 	if sp.Kind != "SelectivePatch" {
// 		t.Fail()
// 	}
// 	if sp.Metadata["name"] != "qliksense-operator-configs" {
// 		t.Fail()
// 	}
// 	if sp.Patches[0].Target.LabelSelector != "app=qliksense" || sp.Patches[0].Target.Kind != "SuperConfigMap" {
// 		t.Fail()
// 	}
// 	scm := &config.SupperConfigMap{
// 		ApiVersion: "qlik.com/v1",
// 		Kind:       "SuperConfigMap",
// 		Metadata: map[string]string{
// 			"name": "qliksense-configs",
// 		},
// 		Data: map[string]string{
// 			"acceptEULA": "yes",
// 		},
// 	}
// 	scm2 := &config.SupperConfigMap{}
// 	yaml.Unmarshal([]byte(sp.Patches[0].Patch), scm2)
// 	if !reflect.DeepEqual(scm, scm2) {
// 		t.Fail()
// 	}
// }

// func TestProcessConfigs(t *testing.T) {
// 	reader := setupCr(t)
// 	cfg, err := config.ReadCRSpecFromFile(reader)
// 	if err != nil {
// 		t.Fatalf("error reading config from file")
// 	}

// 	td, dir := createManifestsStructure(t)

// 	cfg.ManifestsRoot = dir
// 	ProcessConfigs(cfg)
// 	content, _ := ioutil.ReadFile(filepath.Join(dir, ".operator", "configs", "qliksense.yaml"))

// 	sp := getSuperConfigSPTemplate("qliksense")
// 	scm := getSuperConfigMapTemplate("qliksense")
// 	scm.Data = map[string]string{
// 		"acceptEULA": "yes",
// 	}
// 	phb, _ := yaml.Marshal(scm)
// 	sp.Patches = []types.Patch{
// 		types.Patch{
// 			Patch:  string(phb),
// 			Target: getSelector("SuperConfigMap", "qliksense"),
// 		},
// 	}
// 	spOut := &config.SelectivePatch{}
// 	yaml.Unmarshal(content, spOut)
// 	if !reflect.DeepEqual(sp, spOut) {
// 		t.Fail()
// 	}

// 	td()
// }
