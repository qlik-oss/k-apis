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
	cfg.Spec.ManifestsRoot = dir

	storageClassFileName := filepath.Join(cfg.Spec.GetManifestsRoot(), operatorPatchBaseFolder, "transformers", "storage-class.yaml")

	oldCount := strings.Count(getFileContent(storageClassFileName, t), "value: false")
	if oldCount <= 0 {
		t.Log("value: false not found in " + storageClassFileName)
		t.FailNow()
	}
	cfg.Spec.StorageClassName = ""
	err = ProcessStorageClassName(cfg.Spec)

	newCount := strings.Count(getFileContent(storageClassFileName, t), "value: true")

	if newCount != 0 {
		t.Fail()
	}
	cfg.Spec.StorageClassName = "efs"
	err = ProcessStorageClassName(cfg.Spec)

	newCount = strings.Count(getFileContent(storageClassFileName, t), "value: true")

	if newCount != oldCount {
		t.Fail()
	}
	cfg.Spec.StorageClassName = ""

	td()
}
