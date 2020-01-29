package qust

import (
	"github.com/qlik-oss/k-apis/config"
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"
)

func TestProcessNamespace(t *testing.T) {
	// create CR
	reader := setupCr(t)
	cfg, err := config.ReadCRSpecFromFile(reader)
	if err != nil {
		t.Fatalf("error reading config from file")
	}
	// create manifests structure
	td, dir := createManifestsStructure(t)
	cfg.ManifestsRoot = dir
	myNs := "test-ns"
	cfg.NameSpace = myNs

	err = ProcessNamespace(cfg)
	if err != nil {
		td()
		t.FailNow()
	}
	nsFileName := "namespace-" + cfg.NameSpace + ".yaml"
	nsFileFullPath := filepath.Join(cfg.ManifestsRoot, operatorPatchBaseFolder, "transformers", nsFileName)
	kustFile := filepath.Join(cfg.ManifestsRoot, operatorPatchBaseFolder, "transformers", "kustomization.yaml")

	if !strings.Contains(getFileContent(nsFileFullPath, t), myNs) {
		t.Log("Namespace patch file not patch")
		t.Fail()
	}

	if !strings.Contains(getFileContent(kustFile, t), nsFileName) {
		t.Log(nsFileName + " not added in kustomization")
		t.Fail()
	}

	td()
}

func getFileContent(fileName string, t *testing.T) string {
	content, err := ioutil.ReadFile(fileName)
	if err != nil {
		t.Log("Cannot read file " + fileName)
		t.FailNow()
	}
	return string(content)
}
