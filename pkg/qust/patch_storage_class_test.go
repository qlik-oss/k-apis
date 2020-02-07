package qust

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/qlik-oss/k-apis/pkg/config"
)

func TestProcessStorageClassName(t *testing.T) {
	// create CR
	reader := setupCr(t)
	cfg, err := config.ReadCRSpecFromFile(reader)
	if err != nil {
		t.Fatalf("error reading config from file")
	}
	// create manifests structure
	td, dir := createManifestsStructure(t)
	cfg.ManifestsRoot = filepath.Join(dir, "manifests")

	storageClassTemplateName := filepath.Join(cfg.GetManifestsRoot(), operatorPatchBaseFolder, "transformers", "storage-class-template.yaml")


	oldCount := strings.Count(getFileContent(storageClassFileName, t), "value: false")

	if oldCount <= 0 {
		t.Log("value: false not found in " + storageClassTemplateName)
		t.FailNow()
	}
	cfg.StorageClassName = ""
	err = ProcessStorageClassName(cfg)

	newCount := strings.Count(getFileContent(storageClassFileName, t), "value: true")

	if newCount != 0 {
		t.Fail()
	}
	cfg.StorageClassName = "efs"
	err = ProcessStorageClassName(cfg)


	storageClassReleaseName := filepath.Join(cfg.GetManifestsRoot(), operatorPatchBaseFolder, "transformers", cfg.StorageClassName+".yaml")

	newCount = strings.Count(getFileContent(storageClassReleaseName, t), "value: true")


	if newCount != oldCount {
		t.Fail()
	}
	cfg.StorageClassName = ""

	td()
}
