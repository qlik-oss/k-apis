package cr

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/qlik-oss/k-apis/pkg/config"
	"github.com/qlik-oss/k-apis/pkg/git"
	"github.com/qlik-oss/k-apis/pkg/utils"
	"gopkg.in/yaml.v2"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_DeleteKeysClusterBackup_deletesSecret(t *testing.T) {
	if os.Getenv("EXECUTE_K8S_TESTS") != "true" {
		t.SkipNow()
	}

	cr := config.KApiCr{}
	if err := yaml.Unmarshal([]byte(`
apiVersion: qlik.com/v1
kind: Qliksense
metadata:
  name: test-cr
`), &cr); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	kubeconfigPath := filepath.Join(userHomeDir, ".kube", "config")
	if secretsClient, err := utils.GetSecretsClient(kubeconfigPath); err != nil {
		t.Fatalf("unexpected error: %v", err)
	} else if err := secretsClient.Delete("test-cr-operator-state-backup", &metav1.DeleteOptions{}); err != nil && !errors.IsNotFound(err) {
		t.Fatalf("unexpected error: %v\n", err)
	} else if _, err := secretsClient.Create(&v1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-cr-operator-state-backup",
		},
		Type: v1.SecretTypeOpaque,
		Data: map[string][]byte{"foo": []byte("bar")},
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	} else if _, err := secretsClient.Get("test-cr-operator-state-backup", metav1.GetOptions{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	} else if err := DeleteKeysClusterBackup(&cr, kubeconfigPath); err != nil {
		t.Fatalf("unexpected error: %v", err)
	} else if _, err := secretsClient.Get("test-cr-operator-state-backup", metav1.GetOptions{}); err == nil {
		t.Fatal("expected an error, but didn't get it")
	} else if !errors.IsNotFound(err) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func Test_DeleteKeysClusterBackup_forcesKeyRotation(t *testing.T) {
	if os.Getenv("EXECUTE_K8S_TESTS") != "true" {
		t.SkipNow()
	}

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
	secretsClient, err := utils.GetSecretsClient(kubeconfigPath)
	if err != nil {
		t.Fatalf("unexpected error: %v\n", err)
	}

	var backedUpEjsonKeys, backedUpApplicationKeys []byte
	var found bool
	GeneratePatches(&cr, config.KeysActionRestoreOrRotate, kubeconfigPath)
	if secret, err := secretsClient.Get("test-cr-operator-state-backup", metav1.GetOptions{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	} else if backedUpEjsonKeys, found = secret.Data["ejson-keys"]; !found {
		t.Fatalf("expected key: %v to be present in the secret\n", "ejson-keys")
	} else if backedUpApplicationKeys, found = secret.Data["operator-keys"]; !found {
		t.Fatalf("expected key: %v to be present in the secret\n", "operator-keys")
	}

	if err := DeleteKeysClusterBackup(&cr, kubeconfigPath); err != nil {
		t.Fatalf("unexpected error: %v", err)
	} else {
		GeneratePatches(&cr, config.KeysActionRestoreOrRotate, kubeconfigPath)
	}

	if secret, err := secretsClient.Get("test-cr-operator-state-backup", metav1.GetOptions{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	} else if newEjsonKeys, found := secret.Data["ejson-keys"]; !found {
		t.Fatalf("expected key: %v to be present in the secret\n", "ejson-keys")
	} else if bytes.Equal(newEjsonKeys, backedUpEjsonKeys) {
		t.Fatalf("did not expect secret data to equal backed up data for key: %v in the secret\n", "ejson-keys")
	} else if newApplicationKeys, found := secret.Data["operator-keys"]; !found {
		t.Fatalf("expected key: %v to be present in the secret\n", "operator-keys")
	} else if bytes.Equal(newApplicationKeys, backedUpApplicationKeys) {
		t.Fatalf("did not expect secret data to equal backed up data for key: %v in the secret\n", "operator-keys")
	}
}

func Test_DeleteKeysClusterBackup_doesNotThrowErrorsIfSecretNotThere(t *testing.T) {
	if os.Getenv("EXECUTE_K8S_TESTS") != "true" {
		t.SkipNow()
	}

	cr := config.KApiCr{}
	if err := yaml.Unmarshal([]byte(`
apiVersion: qlik.com/v1
kind: Qliksense
metadata:
  name: test-cr
`), &cr); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	kubeconfigPath := filepath.Join(userHomeDir, ".kube", "config")
	if secretsClient, err := utils.GetSecretsClient(kubeconfigPath); err != nil {
		t.Fatalf("unexpected error: %v", err)
	} else if err := secretsClient.Delete("test-cr-operator-state-backup", &metav1.DeleteOptions{}); err != nil && !errors.IsNotFound(err) {
		t.Fatalf("unexpected error: %v\n", err)
	} else if err := DeleteKeysClusterBackup(&cr, kubeconfigPath); err != nil {
		t.Fatalf("unexpected error: %v", err)
	} else if _, err := secretsClient.Get("test-cr-operator-state-backup", metav1.GetOptions{}); err == nil {
		t.Fatal("expected an error, but didn't get it")
	} else if !errors.IsNotFound(err) {
		t.Fatalf("unexpected error: %v", err)
	}
}
