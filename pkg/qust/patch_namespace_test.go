package qust

import (
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"

	"github.com/qlik-oss/k-apis/pkg/config"
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
	cfg.Spec.ManifestsRoot = dir
	myNs := "test-ns"
	cfg.GetObjectMeta().SetNamespace(myNs)

	err = ProcessNamespace(cfg)
	if err != nil {
		td()
		t.FailNow()
	}
	nsFileName := "namespace-" + cfg.GetObjectMeta().GetNamespace() + ".yaml"
	nsFileFullPath := filepath.Join(cfg.Spec.GetManifestsRoot(), operatorPatchBaseFolder, "transformers", nsFileName)
	kustFile := filepath.Join(cfg.Spec.GetManifestsRoot(), operatorPatchBaseFolder, "transformers", "kustomization.yaml")

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
