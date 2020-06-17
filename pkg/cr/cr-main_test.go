package cr

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v2"

	"sigs.k8s.io/kustomize/api/filesys"
	"sigs.k8s.io/kustomize/api/konfig"
	"sigs.k8s.io/kustomize/api/krusty"
	"sigs.k8s.io/kustomize/api/types"

	"github.com/qlik-oss/k-apis/pkg/config"
	"github.com/qlik-oss/k-apis/pkg/git"
)

func TestGeneratePatches(t *testing.T) {
	t.Skip()

	tmpDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatalf("unexpected error: %v\n", err)
	}
	fmt.Printf("--AB: tmpDir: %v\n", tmpDir)
	//defer os.RemoveAll(tmpDir)

	configPath := filepath.Join(tmpDir, "config")
	if repo, err := git.CloneRepository(configPath, "https://github.com/qlik-oss/qliksense-k8s", nil); err != nil {
		t.Fatalf("unexpected error: %v\n", err)
	} else if err := git.Checkout(repo, "update/qliktrial-1.50.3", "update/qliktrial-1.50.3", nil); err != nil {
		t.Fatalf("unexpected error: %v\n", err)
	}

	cr := config.KApiCr{}
	if err := yaml.Unmarshal([]byte(fmt.Sprintf(`
apiVersion: qlik.com/v1
kind: Qliksense
metadata:
  name: test-cr
spec:
  profile: base
  manifestsRoot: %s
  rotateKeys: "None"
  secrets:
    qliksense:
    - name: mongodbUri
      value: mongo://mongo:3307
    collections:
    - name: mongodbUri
      value: mongo://mongo:3308
`, filepath.Join(tmpDir, "config"))), &cr); err != nil {
		t.Fatalf("unexpected error: %v\n", err)
	}

	GeneratePatches(&cr, "won't-use")

	//os.Unsetenv("EJSON_KEY")
	if manifest, err := executeKustomizeBuild(filepath.Join(configPath, "manifests", "base", "resources", "collections")); err != nil {
		t.Fatalf("unexpected kustomize error: %v\n", err)
	} else if err := ioutil.WriteFile(filepath.Join(tmpDir, "manifest.yaml"), manifest, os.ModePerm); err != nil {
		t.Fatalf("unexpected error: %v\n", err)
	}
}

func executeKustomizeBuild(directory string) ([]byte, error) {
	options := &krusty.Options{
		DoLegacyResourceSort: false,
		LoadRestrictions:     types.LoadRestrictionsNone,
		DoPrune:              false,
		PluginConfig:         konfig.DisabledPluginConfig(),
	}
	k := krusty.MakeKustomizer(filesys.MakeFsOnDisk(), options)
	resMap, err := k.Run(directory)
	if err != nil {
		return nil, err
	}
	return resMap.AsYaml()
}
