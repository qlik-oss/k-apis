package qust

import (
	"github.com/qlik-oss/k-apis/config"
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"
)

func TestProcessStorageClassName(t *testing.T) {
	// create CR
	reader := setupCr(t)
	cfg, err := config.ReadCRConfigFromFile(reader)
	if err != nil {
		t.Fatalf("error reading config from file")
	}
	// create manifests structure
	td, dir := createManifestsStructure(t)
	cfg.ManifestsRoot = dir

	storageClassFileName := filepath.Join(cfg.ManifestsRoot, operatorPatchBaseFolder, "transformers", "storage-class.yaml")

	oldCount := strings.Count(getStorageFileContent(storageClassFileName, t), "value: false")
	if oldCount <= 0 {
		t.Log("value: false not found in " + storageClassFileName)
		t.FailNow()
	}
	cfg.StorageClassName = ""
	err = ProcessStorageClassName(cfg)

	newCount := strings.Count(getStorageFileContent(storageClassFileName, t), "value: true")

	if newCount != 0 {
		t.Fail()
	}
	cfg.StorageClassName = "efs"
	err = ProcessStorageClassName(cfg)

	newCount = strings.Count(getStorageFileContent(storageClassFileName, t), "value: true")

	if newCount != oldCount {
		t.Fail()
	}
	cfg.StorageClassName = ""

	td()
}

func getStorageFileContent(fileName string, t *testing.T) string {
	content, err := ioutil.ReadFile(fileName)
	if err != nil {
		t.Log("Cannot read file " + fileName)
		t.FailNow()
	}
	return string(content)
}
