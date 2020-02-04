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
	Directory    string
	ConfigmapKey string
}

const (
	releaseLabelKey      = "release"
	releaseLabelValue    = "qliksense"
	defaultNamespaceName = "default"
)

func Backup(kubeconfigPath, configMapName, namespaceName string, backupDirs []BackupDir) error {
	if namespaceName == "" {
		namespaceName = defaultNamespaceName
	}

	configMapsClient, err := getConfigMapsClient(kubeconfigPath, namespaceName)
	if err != nil {
		return err
	}

	configMapBinaryData, err := getConfigMapBinaryData(backupDirs)
	if err != nil {
		return err
	}

	configMap, err := configMapsClient.Get(configMapName, metaV1.GetOptions{})
	if err != nil && kubeApiErrors.IsNotFound(err) {
		//doesn't exist, create:
		_, err = configMapsClient.Create(&v1.ConfigMap{
			ObjectMeta: metaV1.ObjectMeta{
				Name:      configMapName,
				Namespace: namespaceName,
				Labels:    map[string]string{releaseLabelKey: releaseLabelValue},
			},
			BinaryData: configMapBinaryData,
		})
	} else if err == nil {
		//exists, update:
		configMap.BinaryData = configMapBinaryData
		_, err = configMapsClient.Update(configMap)
	}
	return err
}

func Restore(kubeconfigPath, configMapName, namespaceName string, backupInfos []BackupDir) error {
	if namespaceName == "" {
		namespaceName = defaultNamespaceName
	}

	configMapsClient, err := getConfigMapsClient(kubeconfigPath, namespaceName)
	if err != nil {
		return err
	}

	configMap, err := configMapsClient.Get(configMapName, metaV1.GetOptions{})
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
		archiveFilePath := path.Join(tmpDir, fmt.Sprintf("%v.tar.gz", backupInfo.ConfigmapKey))
		if data, ok := configMap.BinaryData[backupInfo.ConfigmapKey]; !ok {
			return fmt.Errorf("configmap %v in namespace: %v does not have binaryData for key: %v", configMapName, namespaceName, backupInfo.ConfigmapKey)
		} else if err := ioutil.WriteFile(archiveFilePath, data, os.ModePerm); err != nil {
			return err
		} else if err := tarGzArchiver.Unarchive(archiveFilePath, backupInfo.Directory); err != nil {
			return err
		}
	}

	return nil
}

func getConfigMapsClient(kubeconfigPath string, namespaceName string) (clientV1.ConfigMapInterface, error) {
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return nil, err
	}

	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	configMapsClient := clientSet.CoreV1().ConfigMaps(namespaceName)
	return configMapsClient, nil
}

func getConfigMapBinaryData(backupDirs []BackupDir) (map[string][]byte, error) {
	tmpDir, err := ioutil.TempDir("", "")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpDir)

	configMapBinaryData := make(map[string][]byte)
	for _, backupDir := range backupDirs {
		archiveFilePath := path.Join(tmpDir, fmt.Sprintf("%v.tar.gz", backupDir.ConfigmapKey))
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
			configMapBinaryData[backupDir.ConfigmapKey] = data
		}
	}
	return configMapBinaryData, nil
}
