package state

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"

	"github.com/qlik-oss/k-apis/pkg/utils"

	"github.com/mholt/archiver/v3"
	v1 "k8s.io/api/core/v1"
	kubeApiErrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type BackupDir struct {
	Directory string
	Key       string
}

const (
	releaseLabelKey          = "release"
	defaultReleaseLabelValue = "qliksense"
)

func Backup(kubeconfigPath, secretName, releaseLabelValue string, backupDirs []BackupDir) error {
	if releaseLabelValue == "" {
		releaseLabelValue = defaultReleaseLabelValue
	}
	secretsClient, err := utils.GetSecretsClient(kubeconfigPath)
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
				Name:   secretName,
				Labels: map[string]string{releaseLabelKey: releaseLabelValue},
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
	secretsClient, err := utils.GetSecretsClient(kubeconfigPath)
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
			return &kubeApiErrors.StatusError{ErrStatus: metaV1.Status{
				Status:  metaV1.StatusFailure,
				Code:    http.StatusNotFound,
				Reason:  metaV1.StatusReasonNotFound,
				Message: fmt.Sprintf("key: %v not found in secret: %v", backupInfo.Key, secretName),
			}}
		} else if err := ioutil.WriteFile(archiveFilePath, data, os.ModePerm); err != nil {
			return err
		} else if err := tarGzArchiver.Unarchive(archiveFilePath, backupInfo.Directory); err != nil {
			return err
		}
	}

	return nil
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
		if fileInfos, err := ioutil.ReadDir(backupDir.Directory); err != nil {
			return nil, err
		} else {
			for _, fileInfo := range fileInfos {
				archiveSources = append(archiveSources, filepath.Join(backupDir.Directory, fileInfo.Name()))
			}
		}
		if err := archiver.NewTarGz().Archive(archiveSources, archiveFilePath); err != nil {
			return nil, err
		} else if data, err := ioutil.ReadFile(archiveFilePath); err != nil {
			return nil, err
		} else {
			binaryData[backupDir.Key] = data
		}
	}
	return binaryData, nil
}
