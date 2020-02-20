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
	_, dir := createManifestsStructure(t)
	cfg.ManifestsRoot = dir

	releaseFileName := filepath.Join(cfg.GetManifestsRoot(), operatorPatchBaseFolder, "transformers", "new-release.yaml")

	cfg.ReleaseName = "new-release"
	err = ProcessReleaseName(cfg)

	newCount := strings.Count(getFileContent(releaseFileName, t), "release: new-release")
	t.Log(releaseFileName)
	if newCount != 1 {
		t.Fail()
	}

	//td()
}
