package qust

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/qlik-oss/k-apis/pkg/config"
)

func TestProcessReleaseName(t *testing.T) {
	// create CR
	reader := setupCr(t)
	cfg, err := config.ReadCRSpecFromFile(reader)
	if err != nil {
		t.Fatalf("error reading config from file")
	}
	// create manifests structure
	td, dir := createManifestsStructure(t)
	cfg.ManifestsRoot = filepath.Join(dir, "manifests")

	releaseFileName := filepath.Join(cfg.GetManifestsRoot(), operatorPatchBaseFolder, "transformers", "release-name.yaml")

	oldCount := strings.Count(getFileContent(releaseFileName, t), "release: qliksense")
	if oldCount != 1 {
		t.Log("value: false not found in " + releaseFileName)
		t.FailNow()
	}
	cfg.ReleaseName = ""
	err = ProcessReleaseName(cfg)

	newCount := strings.Count(getFileContent(releaseFileName, t), "release: qliksense")

	if newCount != oldCount {
		t.Fail()
	}
	cfg.ReleaseName = "new-release"
	err = ProcessReleaseName(cfg)

	newCount = strings.Count(getFileContent(releaseFileName, t), "release: new-release")

	if newCount != 1 {
		t.Fail()
	}
	cfg.ReleaseName = ""

	td()
}
