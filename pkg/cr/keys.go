package cr

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/qlik-oss/k-apis/pkg/utils"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Shopify/ejson"
	"github.com/qlik-oss/k-apis/pkg/config"
	"github.com/qlik-oss/k-apis/pkg/qust"
	"github.com/qlik-oss/k-apis/pkg/state"
	"k8s.io/apimachinery/pkg/api/errors"
)

func finalizeKeys(cr *config.KApiCr, keysAction config.KeysAction, kubeConfigPath string, ejsonPublicKey string) error {
	if keysAction == config.KeysActionDoNothing {
		log.Println("no keys operations")
		return nil
	}

	keysFound := false
	if keysAction == config.KeysActionRestoreOrRotate {
		if err := state.Restore(kubeConfigPath, getBackupObjectName(cr), cr.GetObjectMeta().GetNamespace(), []state.BackupDir{
			{Key: "operator-keys", Directory: filepath.Join(cr.Spec.GetManifestsRoot(), ".operator/keys")},
		}); err != nil {
			if !errors.IsNotFound(err) {
				return fmt.Errorf("error restoring keys from the cluster: %w", err)
			}
		} else {
			log.Println("restored application keys from the cluster")
			keysFound = true
		}
	}

	if keysAction == config.KeysActionForceRotate || !keysFound {
		if err := qust.GenerateKeys(cr.Spec, ejsonPublicKey); err != nil {
			return fmt.Errorf("error generating application keys: %w", err)
		} else {
			log.Println("generated application keys")
			if err := state.Backup(kubeConfigPath, getBackupObjectName(cr), cr.GetObjectMeta().GetNamespace(), cr.GetName(), []state.BackupDir{
				{Key: "operator-keys", Directory: filepath.Join(cr.Spec.GetManifestsRoot(), ".operator/keys")},
				{Key: "ejson-keys", Directory: getEjsonKeyDir(defaultEjsonKeydir)},
			}); err != nil {
				return fmt.Errorf("error backing up keys to the cluster: %w", err)
			}
			log.Println("backed up application keys to the cluster")
		}
	}

	return nil
}

func extractEjsonKeysFromTheEnvironment() (ejsonPublicKey, ejsonPrivateKey string) {
	if ejsonPrivateKey = os.Getenv("EJSON_KEY"); ejsonPrivateKey != "" {
		ejsonKeyDir := os.Getenv("EJSON_KEYDIR")
		if ejsonKeyDir == "" {
			ejsonKeyDir = defaultEjsonKeydir
		}
		if fileInfos, err := ioutil.ReadDir(ejsonKeyDir); err != nil {
			log.Printf("failed listing the EJSON_KEYDIR: %v\n", ejsonKeyDir)
			return "", ""
		} else {
			for _, fileInfo := range fileInfos {
				if fileInfo.Mode().IsRegular() {
					possibleEjsonPublicKeyFilePath := filepath.Join(ejsonKeyDir, fileInfo.Name())
					if fileContent, err := ioutil.ReadFile(possibleEjsonPublicKeyFilePath); err != nil {
						log.Printf("failed reading file: %v while trying to find the EJSON publicKey\n", possibleEjsonPublicKeyFilePath)
						return "", ""
					} else if strings.TrimSpace(string(fileContent)) == ejsonPrivateKey {
						ejsonPublicKey = fileInfo.Name()
						return ejsonPublicKey, ejsonPrivateKey
					}
				}
			}
		}
	} else {
		ejsonKeyDir := os.Getenv("EJSON_KEYDIR")
		if ejsonKeyDir == "" {
			ejsonKeyDir = defaultEjsonKeydir
		}
		if fileInfos, err := ioutil.ReadDir(ejsonKeyDir); err != nil {
			log.Printf("failed listing the EJSON_KEYDIR: %v\n", ejsonKeyDir)
			return "", ""
		} else if len(fileInfos) == 1 {
			possibleEjsonPublicKeyFilePath := filepath.Join(ejsonKeyDir, fileInfos[0].Name())
			if fileContent, err := ioutil.ReadFile(possibleEjsonPublicKeyFilePath); err != nil {
				log.Printf("failed reading file: %v while trying to find the EJSON publicKey\n", possibleEjsonPublicKeyFilePath)
				return "", ""
			} else {
				ejsonPublicKey = fileInfos[0].Name()
				ejsonPrivateKey = strings.TrimSpace(string(fileContent))
				return ejsonPublicKey, ejsonPrivateKey
			}
		}
	}
	return "", ""
}

func processEjsonKeys(cr *config.KApiCr, keysAction config.KeysAction, kubeConfigPath string, defaultEjsonKeydir string) (ejsonPublicKey string, ejsonPrivateKey string, err error) {
	if keysAction == config.KeysActionDoNothing {
		ejsonPublicKey, ejsonPrivateKey = extractEjsonKeysFromTheEnvironment()
		return ejsonPublicKey, ejsonPrivateKey, nil
	}

	keysFound := false
	if keysAction == config.KeysActionRestoreOrRotate {
		if ejsonPublicKey, ejsonPrivateKey, err = restoreEjsonKeysFromCluster(cr, defaultEjsonKeydir, kubeConfigPath); err != nil {
			if !errors.IsNotFound(err) {
				log.Printf("error restoring the ejson key pair from the cluster: %v\n", err)
				return "", "", err
			}
		} else {
			log.Println("restored ejson keys from the cluster")
			keysFound = true
		}
	}

	if keysAction == config.KeysActionForceRotate || !keysFound {
		if ejsonPublicKey, ejsonPrivateKey, err = ejson.GenerateKeypair(); err != nil {
			log.Printf("error generating an ejson key pair: %v\n", err)
			return "", "", err
		} else if err = rewriteEjsonKeys(defaultEjsonKeydir, ejsonPublicKey, ejsonPrivateKey); err != nil {
			log.Printf("error rewriting ejson keys: %v\n", err)
			return "", "", err
		}
	}

	return ejsonPublicKey, ejsonPrivateKey, err
}

func getBackupObjectName(cr *config.KApiCr) string {
	return fmt.Sprintf("%s-%s", cr.GetName(), defaultBackupObjectName)
}

func getEjsonKeyDir(defaultKeyDir string) string {
	ejsonKeyDir := os.Getenv("EJSON_KEYDIR")
	if ejsonKeyDir == "" {
		ejsonKeyDir = defaultKeyDir
	}
	return ejsonKeyDir
}

func rewriteEjsonKeys(defaultKeyDir, ejsonPublicKey, ejsonPrivateKey string) error {
	keyDir := getEjsonKeyDir(defaultKeyDir)
	keyPath := path.Join(keyDir, ejsonPublicKey)
	if err := os.MkdirAll(keyDir, os.ModePerm); err != nil {
		log.Printf("error makeing sure private key storage directory: %v exists, error: %v\n", keyDir, err)
	} else if err := cleanEjsonKeysDir(keyDir); err != nil {
		log.Printf("error cleaning key directory: %v, error: %v\n", keyDir, err)
		return err
	} else if err := ioutil.WriteFile(keyPath, []byte(ejsonPrivateKey), os.ModePerm); err != nil {
		log.Printf("error storing writing ejson key file: %v, error: %v\n", keyPath, err)
		return err
	}
	return nil
}

func cleanEjsonKeysDir(keyDir string) error {
	if dirItems, err := ioutil.ReadDir(keyDir); err != nil {
		log.Printf("error reading key directory: %v, error: %v\n", keyDir, err)
		return err
	} else {
		for _, d := range dirItems {
			keyFile := path.Join(keyDir, d.Name())
			if err := os.RemoveAll(keyFile); err != nil {
				log.Printf("error deleting ejson key file: %v, error: %v\n", keyFile, err)
				return err
			}
		}
		return nil
	}
}

func restoreEjsonKeysFromCluster(cr *config.KApiCr, defaultKeyDir string, kubeConfigPath string) (ejsonPublicKey, ejsonPrivateKey string, err error) {
	if err := state.Restore(kubeConfigPath, getBackupObjectName(cr), cr.GetObjectMeta().GetNamespace(), []state.BackupDir{
		{Key: "ejson-keys", Directory: getEjsonKeyDir(defaultKeyDir)},
	}); err != nil {
		return "", "", err
	} else {
		return loadEjsonKeysFromKeyDir(defaultKeyDir)
	}
}

func loadEjsonKeysFromKeyDir(defaultKeyDir string) (ejsonPublicKey, ejsonPrivateKey string, err error) {
	keyDir := getEjsonKeyDir(defaultKeyDir)
	if dirItems, err := ioutil.ReadDir(keyDir); err != nil {
		log.Printf("error reading key directory: %v, error: %v\n", keyDir, err)
		return "", "", err
	} else {
		for _, fileInfo := range dirItems {
			if fileInfo.IsDir() {
				continue
			} else if ejsonPrivateKeyBytes, err := ioutil.ReadFile(path.Join(keyDir, fileInfo.Name())); err != nil {
				return "", "", err
			} else {
				return fileInfo.Name(), string(ejsonPrivateKeyBytes), nil
			}
		}
		err = fmt.Errorf("no ejson keys found in directory: %v", keyDir)
		log.Printf("%v\n", err)
		return "", "", err
	}
}

func DeleteKeysClusterBackup(cr *config.KApiCr, kubeConfigPath string) error {
	if secretsClient, err := utils.GetSecretsClient(kubeConfigPath, cr.GetObjectMeta().GetNamespace()); err != nil {
		return err
	} else if err := secretsClient.Delete(context.TODO(), getBackupObjectName(cr), metaV1.DeleteOptions{}); err != nil && !errors.IsNotFound(err) {
		return err
	}
	return nil
}
