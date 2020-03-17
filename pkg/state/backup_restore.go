package state

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	"github.com/mholt/archiver/v3"
	v1 "k8s.io/api/core/v1"
	kubeApiErrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	clientV1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/clientcmd"
)

type BackupDir struct {
	Directory string
	Key       string
}

const (
	releaseLabelKey          = "release"
	defaultReleaseLabelValue = "qliksense"
	defaultNamespaceName     = "default"
)

func Backup(kubeconfigPath, secretName, namespaceName, releaseLabelValue string, backupDirs []BackupDir) error {
	if namespaceName == "" {
		namespaceName = defaultNamespaceName
	}
	if releaseLabelValue == "" {
		releaseLabelValue = defaultReleaseLabelValue
	}
	secretsClient, err := getSecretsClient(kubeconfigPath, namespaceName)
	if err != nil {
		return err
	}

	binaryData, err := getBinaryData(backupDirs)
	if err != nil {
		return err
	}

	secret, err := secretsClient.Get(secretName, metaV1.GetOptions{})
	if err != nil && kubeApiErrors.IsNotFound(err) {
		//doesn't exist, create:
		_, err = secretsClient.Create(&v1.Secret{
			ObjectMeta: metaV1.ObjectMeta{
				Name:      secretName,
				Namespace: namespaceName,
				Labels:    map[string]string{releaseLabelKey: releaseLabelValue},
			},
			Data: binaryData,
		})
	} else if err == nil {
		//exists, update:
		secret.Data = binaryData
		_, err = secretsClient.Update(secret)
	}
	return err
}

func Restore(kubeconfigPath, secretName, namespaceName string, backupInfos []BackupDir) error {
	if namespaceName == "" {
		namespaceName = defaultNamespaceName
	}

	secretsClient, err := getSecretsClient(kubeconfigPath, namespaceName)
	if err != nil {
		return err
	}

	secret, err := secretsClient.Get(secretName, metaV1.GetOptions{})
	if err != nil {
		return err
	}

	tmpDir, err := ioutil.TempDir("", "")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	tarGzArchiver := archiver.NewTarGz()
	tarGzArchiver.OverwriteExisting = true

	for _, backupInfo := range backupInfos {
		archiveFilePath := path.Join(tmpDir, fmt.Sprintf("%v.tar.gz", backupInfo.Key))
		if data, ok := secret.Data[backupInfo.Key]; !ok {
			return fmt.Errorf("secret %v in namespace: %v does not have binaryData for key: %v", secretName, namespaceName, backupInfo.Key)
		} else if err := ioutil.WriteFile(archiveFilePath, data, os.ModePerm); err != nil {
			return err
		} else if err := tarGzArchiver.Unarchive(archiveFilePath, backupInfo.Directory); err != nil {
			return err
		}
	}

	return nil
}

func getSecretsClient(kubeconfigPath string, namespaceName string) (clientV1.SecretInterface, error) {
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return nil, err
	}

	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	secretsClient := clientSet.CoreV1().Secrets(namespaceName)
	return secretsClient, nil
}

func getBinaryData(backupDirs []BackupDir) (map[string][]byte, error) {
	tmpDir, err := ioutil.TempDir("", "")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpDir)

	binaryData := make(map[string][]byte)
	for _, backupDir := range backupDirs {
		archiveFilePath := path.Join(tmpDir, fmt.Sprintf("%v.tar.gz", backupDir.Key))
		var archiveSources []string
		if err := filepath.Walk(backupDir.Directory, func(fpath string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if fpath != backupDir.Directory {
				archiveSources = append(archiveSources, fpath)
			}
			return nil
		}); err != nil {
			return nil, err
		} else if err := archiver.NewTarGz().Archive(archiveSources, archiveFilePath); err != nil {
			return nil, err
		} else if data, err := ioutil.ReadFile(archiveFilePath); err != nil {
			return nil, err
		} else {
			binaryData[backupDir.Key] = data
		}
	}
	return binaryData, nil
}
