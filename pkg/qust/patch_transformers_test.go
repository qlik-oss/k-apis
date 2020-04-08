package qust

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/qlik-oss/k-apis/pkg/config"
	"github.com/qlik-oss/k-apis/pkg/git"
)

func TestProcessTransfomer(t *testing.T) {
	tempDir, err := downloadQliksenseK8sForTest()
	if err != nil {
		t.Log(err)
		t.FailNow()
	}
	reader := setupCr(t)
	cfg, err := config.ReadCRSpecFromFile(reader)
	if err != nil {
		t.Fatalf("error reading config from file")
	}
	cfg.Spec.ManifestsRoot = tempDir
	cfg.Spec.AddToSecrets("qliksense", "caCertificates", "somethign", "")
	cfg.Spec.AddToSecrets("audit", "caCertificates", "somethign", "")
	if err := ProcessTransfomer(cfg.Spec); err != nil {
		t.Log(err)
		t.FailNow()
	}
	genTranPath := filepath.Join(tempDir, ".operator", "transformers")
	kFile := filepath.Join(genTranPath, "kustomization.yaml")
	list, err := getResourcesList(kFile)
	if err != nil {
		t.Log(err)
		t.Fail()
	}
	if !contains(list, "qliksense.yaml") {
		t.Log("expected resources is not created")
		t.Logf("%v", list)
		t.Fail()
	}
	if !contains(list, "audit.yaml") {
		t.Log("expected resources is not created")
		t.Logf("%v", list)
		t.Fail()
	}
	bt, err := ioutil.ReadFile(filepath.Join(genTranPath, "qliksense.yaml"))
	if err != nil {
		t.Log(err)
		t.Fail()
	}
	if !strings.Contains(string(bt), "caCertificates") {
		t.Log(string(bt))
		t.Fail()
	}
	if strings.Contains(string(bt), "labelSelector") {
		t.Log(string(bt))
		t.Fail()
	}

	bt, err = ioutil.ReadFile(filepath.Join(genTranPath, "audit.yaml"))
	if err != nil {
		t.Log(err)
		t.Fail()
	}
	if !strings.Contains(string(bt), "caCertificates") {
		t.Log(string(bt))
		t.Fail()
	}
	if !strings.Contains(string(bt), "labelSelector: app=audit") {
		t.Log(string(bt))
		t.Fail()
	}
}

func downloadQliksenseK8sForTest() (string, error) {
	tempDir, err := ioutil.TempDir("", "")
	if err != nil {
		return "", err
	}

	if repo, err := git.CloneRepository(tempDir, "https://github.com/qlik-oss/qliksense-k8s", nil); err != nil {
		return "", err
	} else if err = git.Checkout(repo, "master", fmt.Sprintf("%v-by-operator-%v", "master", uuid.New().String()), nil); err != nil {
		return "", err
	}
	return tempDir, nil
}

func TestLoadExistingOrCreateEmptySelectivePatch(t *testing.T) {
	tempDir, _ := downloadQliksenseK8sForTest()
	t.Log(tempDir)
	_, err := loadExistingOrCreateEmptySelectivePatch("qliksense", "my-patch", filepath.Join(tempDir, ".operator", "transformers"))
	if err != nil {
		t.Log(err)
		t.FailNow()
	}
	if err := writeTranasformer(filepath.Join(tempDir, ".operator", "transformers"), "qliksense", "caCertificates"); err != nil {
		t.Log(err)
		t.Fail()
	}

}
