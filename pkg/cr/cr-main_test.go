package cr

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/qlik-oss/k-apis/pkg/utils"

	"github.com/Shopify/ejson"

	"github.com/qlik-oss/k-apis/pkg/qust"
	"github.com/qlik-oss/k-apis/pkg/state"

	"k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientV1 "k8s.io/client-go/kubernetes/typed/core/v1"

	"gopkg.in/yaml.v2"

	"github.com/otiai10/copy"
	"github.com/qlik-oss/k-apis/pkg/config"
	"github.com/qlik-oss/k-apis/pkg/git"
)

func TestGeneratePatches_acceptEula(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatalf("unexpected error: %v\n", err)
	}
	defer os.RemoveAll(tmpDir)

	configPath := filepath.Join(tmpDir, "config")
	if repo, err := git.CloneRepository(configPath, "https://github.com/qlik-oss/qliksense-k8s", nil); err != nil {
		t.Fatalf("unexpected error: %v\n", err)
	} else if err := git.Checkout(repo, "v1.50.3", "", nil); err != nil {
		t.Fatalf("unexpected error: %v\n", err)
	}

	cr := config.KApiCr{}
	if err := yaml.Unmarshal([]byte(fmt.Sprintf(`
apiVersion: qlik.com/v1
kind: Qliksense
metadata:
  name: test-cr
spec:
  manifestsRoot: %s
  configs:
    qliksense:
    - name: acceptEULA
      value: "yes"
`, filepath.Join(tmpDir, "config"))), &cr); err != nil {
		t.Fatalf("unexpected error: %v\n", err)
	}

	GeneratePatches(&cr, config.KeysActionDoNothing, "won't-use")

	expectedQliksenseAcceptEulePatchYaml := `apiVersion: qlik.com/v1
kind: SelectivePatch
metadata:
  name: qliksense-generated-operator-configs
enabled: true
patches:
- patch: |
    apiVersion: qlik.com/v1
    kind: SuperConfigMap
    metadata:
      name: qliksense-configs
    data:
      acceptEULA: "yes"
  target:
    kind: SuperConfigMap
    labelSelector: app=qliksense
`
	if qliksenseConfigYaml, err := ioutil.ReadFile(filepath.Join(configPath, ".operator", "configs", "qliksense.yaml")); err != nil {
		t.Fatalf("unexpected error: %v\n", err)
	} else if string(qliksenseConfigYaml) != expectedQliksenseAcceptEulePatchYaml {
		t.Fatalf("expected: %v, but got: %v\n", expectedQliksenseAcceptEulePatchYaml, string(qliksenseConfigYaml))
	}

	expectedConfigsKustomizationYaml := `kind: Kustomization
apiVersion: kustomize.config.k8s.io/v1beta1
resources:
- qliksense.yaml
`
	if configsKustomizationYaml, err := ioutil.ReadFile(filepath.Join(configPath, ".operator", "configs", "kustomization.yaml")); err != nil {
		t.Fatalf("unexpected error: %v\n", err)
	} else if string(configsKustomizationYaml) != expectedConfigsKustomizationYaml {
		t.Fatalf("expected: %v, but got: %v\n", expectedConfigsKustomizationYaml, string(configsKustomizationYaml))
	}
}

func TestGeneratePatches_KeysAction(t *testing.T) {
	if os.Getenv("EXECUTE_K8S_TESTS") != "true" {
		t.SkipNow()
	}

	type keysActionTestCaseT struct {
		name            string
		keysAction      config.KeysAction
		setup           func(t *testing.T, secretsClient clientV1.SecretInterface, tmpDir string, cr *config.KApiCr)
		checkAssertions func(t *testing.T, secretsClient clientV1.SecretInterface, tmpDir string, cr *config.KApiCr)
	}
	testCases := []keysActionTestCaseT{
		{
			name:       "KeysActionForceRotate",
			keysAction: config.KeysActionForceRotate,
			setup: func(t *testing.T, secretsClient clientV1.SecretInterface, tmpDir string, cr *config.KApiCr) {
				if err := secretsClient.Delete("test-cr-operator-state-backup", &metaV1.DeleteOptions{}); err != nil && !errors.IsNotFound(err) {
					t.Fatalf("unexpected error: %v\n", err)
				}
			},
			checkAssertions: func(t *testing.T, secretsClient clientV1.SecretInterface, tmpDir string, cr *config.KApiCr) {
				secret, err := secretsClient.Get("test-cr-operator-state-backup", metaV1.GetOptions{})
				if err != nil {
					t.Fatalf("unexpected error: %v\n", err)
				}

				if _, ok := secret.Data["ejson-keys"]; !ok {
					t.Fatalf("expected key: %v to be present in the secret\n", "ejson-keys")
				}

				if _, ok := secret.Data["operator-keys"]; !ok {
					t.Fatalf("expected key: %v to be present in the secret\n", "operator-keys")
				}
			},
		},
		func() keysActionTestCaseT {
			var expectedEjsonKeysK8sData, expectedApplicationKeysK8sData []byte
			var operatorKeysArchiveBackupDirPath string
			return keysActionTestCaseT{
				name:       "KeysActionForceRotate-overwrites",
				keysAction: config.KeysActionForceRotate,
				setup: func(t *testing.T, secretsClient clientV1.SecretInterface, tmpDir string, cr *config.KApiCr) {
					userHomeDir, err := os.UserHomeDir()
					if err != nil {
						t.Fatalf("unexpected error: %v\n", err)
					}

					kubeconfigPath := filepath.Join(userHomeDir, ".kube", "config")
					if ejsonPublicKey, ejsonPrivateKey, err := ejson.GenerateKeypair(); err != nil {
						t.Fatalf("unexpected error: %v\n", err)
					} else if err = rewriteEjsonKeys(filepath.Join(tmpDir, "ejson-keys"), ejsonPublicKey, ejsonPrivateKey); err != nil {
						t.Fatalf("unexpected error: %v\n", err)
					} else if err := qust.GenerateKeys(cr.Spec, ejsonPublicKey); err != nil {
						t.Fatalf("unexpected error: %v\n", err)
					} else {
						if err := state.Backup(kubeconfigPath, getBackupObjectName(cr), cr.GetObjectMeta().GetNamespace(), cr.GetName(), []state.BackupDir{
							{Key: "operator-keys", Directory: filepath.Join(cr.Spec.GetManifestsRoot(), ".operator/keys")},
							{Key: "ejson-keys", Directory: filepath.Join(tmpDir, "ejson-keys")},
						}); err != nil {
							t.Fatalf("unexpected error: %v\n", err)
						}
					}

					operatorKeysArchiveBackupDirPath = filepath.Join(tmpDir, "keys_backup")
					if err := copy.Copy(filepath.Join(cr.Spec.ManifestsRoot, ".operator", "keys"), operatorKeysArchiveBackupDirPath); err != nil {
						t.Fatalf("unexpected error: %v\n", err)
					}

					if r, err := git.OpenRepository(cr.Spec.ManifestsRoot); err != nil {
						t.Fatalf("unexpected error: %v\n", err)
					} else if err := git.DiscardAllUnstagedChanges(r); err != nil {
						t.Fatalf("unexpected error: %v\n", err)
					}

					secret, err := secretsClient.Get("test-cr-operator-state-backup", metaV1.GetOptions{})
					if err != nil {
						t.Fatalf("unexpected error: %v\n", err)
					}
					var foundKey bool
					if expectedEjsonKeysK8sData, foundKey = secret.Data["ejson-keys"]; !foundKey {
						t.Fatalf("expected key: %v to be present in the secret\n", "ejson-keys")
					}
					if expectedApplicationKeysK8sData, foundKey = secret.Data["operator-keys"]; !foundKey {
						t.Fatalf("expected key: %v to be present in the secret\n", "operator-keys")
					}
				},
				checkAssertions: func(t *testing.T, secretsClient clientV1.SecretInterface, tmpDir string, cr *config.KApiCr) {
					secret, err := secretsClient.Get("test-cr-operator-state-backup", metaV1.GetOptions{})
					if err != nil {
						t.Fatalf("unexpected error: %v\n", err)
					}

					if newEjsonKeysK8sData, ok := secret.Data["ejson-keys"]; !ok {
						t.Fatalf("expected key: %v to be present in the secret\n", "ejson-keys")
					} else if bytes.Equal(newEjsonKeysK8sData, expectedEjsonKeysK8sData) {
						t.Fatalf("did not expect data to equal for key: %v in the secret\n", "ejson-keys")
					}

					if newApplicationKeysK8sData, ok := secret.Data["operator-keys"]; !ok {
						t.Fatalf("expected key: %v to be present in the secret\n", "operator-keys")
					} else if bytes.Equal(newApplicationKeysK8sData, expectedApplicationKeysK8sData) {
						t.Fatalf("did not expect data to equal for key: %v in the secret\n", "ejson-keys")
					}

					if equal, err := directoryContentsEqual(filepath.Join(cr.Spec.ManifestsRoot, ".operator", "keys"), operatorKeysArchiveBackupDirPath); err != nil {
						t.Fatalf("unexpected error: %v\n", err)
					} else if equal {
						t.Fatal("did not expect .operator/keys folder to be the same as before keys were rotated, but it was")
					}

					if sameFiles, err := directoryContentsHaveSameFiles(filepath.Join(cr.Spec.ManifestsRoot, ".operator", "keys"), operatorKeysArchiveBackupDirPath); err != nil {
						t.Fatalf("unexpected error: %v\n", err)
					} else if !sameFiles {
						t.Fatal("expect .operator/keys folder to have the same files as before keys were rotated, but it didn't")
					}
				},
			}
		}(),
		{
			name:       "KeysActionRestoreOrRotate-can-behave-like-KeysActionForceRotate",
			keysAction: config.KeysActionRestoreOrRotate,
			setup: func(t *testing.T, secretsClient clientV1.SecretInterface, tmpDir string, cr *config.KApiCr) {
				if err := secretsClient.Delete("test-cr-operator-state-backup", &metaV1.DeleteOptions{}); err != nil && !errors.IsNotFound(err) {
					t.Fatalf("unexpected error: %v\n", err)
				}
			},
			checkAssertions: func(t *testing.T, secretsClient clientV1.SecretInterface, tmpDir string, cr *config.KApiCr) {
				secret, err := secretsClient.Get("test-cr-operator-state-backup", metaV1.GetOptions{})
				if err != nil {
					t.Fatalf("unexpected error: %v\n", err)
				}

				if _, ok := secret.Data["ejson-keys"]; !ok {
					t.Fatalf("expected key: %v to be present in the secret\n", "ejson-keys")
				}

				if _, ok := secret.Data["operator-keys"]; !ok {
					t.Fatalf("expected key: %v to be present in the secret\n", "operator-keys")
				}
			},
		},
		func() keysActionTestCaseT {
			var expectedEjsonKeysK8sData, expectedApplicationKeysK8sData []byte
			var operatorKeysArchiveBackupDirPath string
			return keysActionTestCaseT{
				name:       "KeysActionRestoreOrRotate-restores",
				keysAction: config.KeysActionRestoreOrRotate,
				setup: func(t *testing.T, secretsClient clientV1.SecretInterface, tmpDir string, cr *config.KApiCr) {
					userHomeDir, err := os.UserHomeDir()
					if err != nil {
						t.Fatalf("unexpected error: %v\n", err)
					}

					kubeconfigPath := filepath.Join(userHomeDir, ".kube", "config")
					if ejsonPublicKey, ejsonPrivateKey, err := ejson.GenerateKeypair(); err != nil {
						t.Fatalf("unexpected error: %v\n", err)
					} else if err = rewriteEjsonKeys(filepath.Join(tmpDir, "ejson-keys"), ejsonPublicKey, ejsonPrivateKey); err != nil {
						t.Fatalf("unexpected error: %v\n", err)
					} else if err := qust.GenerateKeys(cr.Spec, ejsonPublicKey); err != nil {
						t.Fatalf("unexpected error: %v\n", err)
					} else {
						if err := state.Backup(kubeconfigPath, getBackupObjectName(cr), cr.GetObjectMeta().GetNamespace(), cr.GetName(), []state.BackupDir{
							{Key: "operator-keys", Directory: filepath.Join(cr.Spec.GetManifestsRoot(), ".operator/keys")},
							{Key: "ejson-keys", Directory: filepath.Join(tmpDir, "ejson-keys")},
						}); err != nil {
							t.Fatalf("unexpected error: %v\n", err)
						}
					}

					operatorKeysArchiveBackupDirPath = filepath.Join(tmpDir, "keys_backup")
					if err := copy.Copy(filepath.Join(cr.Spec.ManifestsRoot, ".operator", "keys"), operatorKeysArchiveBackupDirPath); err != nil {
						t.Fatalf("unexpected error: %v\n", err)
					}

					if r, err := git.OpenRepository(cr.Spec.ManifestsRoot); err != nil {
						t.Fatalf("unexpected error: %v\n", err)
					} else if err := git.DiscardAllUnstagedChanges(r); err != nil {
						t.Fatalf("unexpected error: %v\n", err)
					}

					secret, err := secretsClient.Get("test-cr-operator-state-backup", metaV1.GetOptions{})
					if err != nil {
						t.Fatalf("unexpected error: %v\n", err)
					}
					var foundKey bool
					if expectedEjsonKeysK8sData, foundKey = secret.Data["ejson-keys"]; !foundKey {
						t.Fatalf("expected key: %v to be present in the secret\n", "ejson-keys")
					}
					if expectedApplicationKeysK8sData, foundKey = secret.Data["operator-keys"]; !foundKey {
						t.Fatalf("expected key: %v to be present in the secret\n", "operator-keys")
					}
				},
				checkAssertions: func(t *testing.T, secretsClient clientV1.SecretInterface, tmpDir string, cr *config.KApiCr) {
					secret, err := secretsClient.Get("test-cr-operator-state-backup", metaV1.GetOptions{})
					if err != nil {
						t.Fatalf("unexpected error: %v\n", err)
					}

					if newEjsonKeysK8sData, ok := secret.Data["ejson-keys"]; !ok {
						t.Fatalf("expected key: %v to be present in the secret\n", "ejson-keys")
					} else if !bytes.Equal(newEjsonKeysK8sData, expectedEjsonKeysK8sData) {
						t.Fatalf("expected data does not equal actual data for key: %v in the secret\n", "ejson-keys")
					}

					if newApplicationKeysK8sData, ok := secret.Data["operator-keys"]; !ok {
						t.Fatalf("expected key: %v to be present in the secret\n", "operator-keys")
					} else if !bytes.Equal(newApplicationKeysK8sData, expectedApplicationKeysK8sData) {
						t.Fatalf("expected data does not equal actual data for key: %v in the secret\n", "ejson-keys")
					}

					if equal, err := directoryContentsEqual(filepath.Join(cr.Spec.ManifestsRoot, ".operator", "keys"), operatorKeysArchiveBackupDirPath); err != nil {
						t.Fatalf("unexpected error: %v\n", err)
					} else if !equal {
						t.Fatal("expected .operator/keys folder to be the same as before restore, but it wasn't")
					}
				},
			}
		}(),
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			tmpDir, err := ioutil.TempDir("", "")
			if err != nil {
				t.Fatalf("unexpected error: %v\n", err)
			}
			defer os.RemoveAll(tmpDir)

			if err := os.Mkdir(filepath.Join(tmpDir, "ejson-keys"), os.ModePerm); err != nil {
				t.Fatalf("unexpected error: %v\n", err)
			} else if err := os.Setenv("EJSON_KEYDIR", filepath.Join(tmpDir, "ejson-keys")); err != nil {
				t.Fatalf("unexpected error: %v\n", err)
			}

			configPath := filepath.Join(tmpDir, "config")
			if repo, err := git.CloneRepository(configPath, "https://github.com/qlik-oss/qliksense-k8s", nil); err != nil {
				t.Fatalf("unexpected error: %v\n", err)
			} else if err := git.Checkout(repo, "v1.50.3", "", nil); err != nil {
				t.Fatalf("unexpected error: %v\n", err)
			}

			cr := config.KApiCr{}
			if err := yaml.Unmarshal([]byte(fmt.Sprintf(`
apiVersion: qlik.com/v1
kind: Qliksense
metadata:
  name: test-cr
spec:
  manifestsRoot: %s
`, configPath)), &cr); err != nil {
				t.Fatalf("unexpected error: %v\n", err)
			}

			userHomeDir, err := os.UserHomeDir()
			if err != nil {
				t.Fatalf("unexpected error: %v\n", err)
			}

			kubeconfigPath := filepath.Join(userHomeDir, ".kube", "config")
			secretsClient, err := utils.GetSecretsClient(kubeconfigPath, cr.GetObjectMeta().GetNamespace())
			if err != nil {
				t.Fatalf("unexpected error: %v\n", err)
			}

			testCase.setup(t, secretsClient, tmpDir, &cr)
			GeneratePatches(&cr, testCase.keysAction, kubeconfigPath)
			testCase.checkAssertions(t, secretsClient, tmpDir, &cr)
		})
	}
}

func directoryContentsEqual(dir1 string, dir2 string) (bool, error) {
	if map1, err := getDirMap(dir1); err != nil {
		return false, err
	} else if map2, err := getDirMap(dir2); err != nil {
		return false, err
	} else if !reflect.DeepEqual(map1, map2) {
		return false, nil
	}
	return true, nil
}

func directoryContentsHaveSameFiles(dir1 string, dir2 string) (bool, error) {
	if map1, err := getDirMap(dir1); err != nil {
		return false, err
	} else if map2, err := getDirMap(dir2); err != nil {
		return false, err
	} else {
		for key1 := range map1 {
			if _, ok := map2[key1]; !ok {
				return false, nil
			}
		}
		for key2 := range map2 {
			if _, ok := map1[key2]; !ok {
				return false, nil
			}
		}
	}
	return true, nil
}

func getDirMap(dir string) (map[string][]byte, error) {
	dirMap := make(map[string][]byte)
	if err := filepath.Walk(dir, func(fpath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if fpath != dir && !info.IsDir() {
			if fileContent, err := ioutil.ReadFile(fpath); err != nil {
				return err
			} else {
				dirMap[path.Base(fpath)] = fileContent
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return dirMap, nil
}
