package cr

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/Shopify/ejson"
	"github.com/qlik-oss/k-apis/pkg/config"
	"github.com/qlik-oss/k-apis/pkg/qust"
	"github.com/qlik-oss/k-apis/pkg/state"
)

const (
	defaultEjsonKeydir      = "/opt/ejson/keys"
	defaultBackupObjectName = "operator-state-backup"
)

func GeneratePatches(cr *config.KApiCr, kubeConfigPath string) {
	if err := createPatches(cr, kubeConfigPath); err != nil {
		log.Printf("error creating patches: %v\n", err)
	}
}

/**
// keeping for future code reference
else {
		var r *git.Repository
		var auth transport.AuthMethod
		if cr.Spec.Git.UserName != "" && cr.Spec.Git.Password != "" {
			auth = &http.BasicAuth{
				Username: cr.Spec.Git.UserName,
				Password: cr.Spec.Git.Password,
			}
		}
		if cr.Spec.Git.AccessToken != "" {
			username := cr.Spec.Git.UserName
			if username == "" {
				username = "installer"
			}
			auth = &http.BasicAuth{
				Username: username,
				Password: cr.Spec.Git.AccessToken,
			}
		}

		// Clone or open
		if _, err := os.Stat(cr.Spec.GetManifestsRoot()); os.IsNotExist(err) {
			r, err = crGit.CloneRepository(cr.Spec.GetManifestsRoot(), cr.Spec.Git.Repository, auth)
			if err != nil {
				log.Printf("error cloning repository %s: %v\n", cr.Spec.Git.Repository, err)
				return
			}
		} else {
			r, err = crGit.OpenRepository(cr.Spec.GetManifestsRoot())
			if err != nil {
				log.Printf("error opening repository %s: %v\n", cr.Spec.Git.Repository, err)
				return
			}
		}
		//set reference to head
		headRef, err := r.Head()
		if err != nil {
			log.Printf("error seeting reference: %v\n", err)
			return
		}

		toBranch := fmt.Sprintf("pr-branch-%s", crGit.TokenGenerator())
		//checkout to new branch
		err = crGit.Checkout(r, headRef.Hash().String(), toBranch, auth)
		if err != nil {
			log.Printf("error checking out to %s: %v\n", toBranch, err)
		}

		err = createPatches(cr, kubeConfigPath)
		if err != nil {
			log.Printf("error creating patches: %v\n", err)
			return
		}
		//commit patches
		err = crGit.AddCommit(r, cr.Spec.Git.UserName)
		if err != nil {
			log.Printf("error adding commit: %v\n", err)
			return
		}
		//push patches
		err = crGit.Push(r, auth)
		if err != nil {
			log.Printf("error pushing to %s: %v\n", cr.Spec.Git.Repository, err)
			return
		}
		//create pr
		// err = crGit.CreatePR(cr.GetManifestsRoot(), cr.Git.AccessToken, cr.Git.UserName, toBranch)
		// if err != nil {
		// 	log.Printf("error creating pr against %s: %v\n", cr.Git.Repository, err)

		// }
	}
*/
func createPatches(cr *config.KApiCr, kubeConfigPath string) error {
	//process cr.releaseName
	if err := qust.ProcessReleaseName(cr); err != nil {
		return err
	}
	// process cr.storageClassName
	if cr.Spec.StorageClassName != "" {
		// added to the configs so that down the road it is being processed
		cr.Spec.AddToConfigs("qliksense", "storageClassName", cr.Spec.StorageClassName)
	}
	// process cr.Namespace
	if err := qust.ProcessNamespace(cr); err != nil {
		return err
	}

	// Process cr.configs
	if err := qust.ProcessConfigs(cr.Spec); err != nil {
		return err
	}

	// regenerate ejson key pair or restore it from cluster
	ejsonPublicKey, _, err := processEjsonKeys(cr, defaultEjsonKeydir, kubeConfigPath)
	if err != nil {
		return err
	}

	// Process cr.secrets
	if err := qust.ProcessSecrets(cr.Spec, ejsonPublicKey); err != nil {
		return err
	}

	// patch transformers based on configs and secrets
	if err := qust.ProcessTransfomer(cr.Spec); err != nil {
		return err
	}

	// rotate all application keys and back them up to cluster (also backup the ejson key pair)
	// OR restore all application keys from cluster
	if err := finalizeKeys(cr, kubeConfigPath, ejsonPublicKey); err != nil {
		return err
	}

	return nil
}

func finalizeKeys(cr *config.KApiCr, kubeConfigPath string, ejsonPublicKey string) error {
	if cr.Spec.RotateKeys == "yes" {
		if err := qust.GenerateKeys(cr.Spec, ejsonPublicKey); err != nil {
			return errors.Wrap(err, "error generating application keys")
		} else {
			log.Println("backing up keys to the cluster")
			if err := state.Backup(kubeConfigPath, getBackupObjectName(cr), cr.GetObjectMeta().GetNamespace(), cr.GetName(), []state.BackupDir{
				{Key: "operator-keys", Directory: filepath.Join(cr.Spec.GetManifestsRoot(), ".operator/keys")},
				{Key: "ejson-keys", Directory: getEjsonKeyDir(defaultEjsonKeydir)},
			}); err != nil {
				return errors.Wrap(err, "error backing up keys to the cluster")
			}
		}
	} else if cr.Spec.RotateKeys == "no" {
		log.Println("restoring keys from the cluster")
		if err := state.Restore(kubeConfigPath, getBackupObjectName(cr), cr.GetObjectMeta().GetNamespace(), []state.BackupDir{
			{Key: "operator-keys", Directory: filepath.Join(cr.Spec.GetManifestsRoot(), ".operator/keys")},
		}); err != nil {
			return errors.Wrap(err, "error restoring keys from the cluster")
		}
	} else if cr.Spec.RotateKeys == "None" {
		log.Println("no keys operations")
	}
	return nil
}

func processEjsonKeys(cr *config.KApiCr, defaultEjsonKeydir string, kubeConfigPath string) (ejsonPublicKey, ejsonPrivateKey string, err error) {
	if cr.Spec.RotateKeys == "yes" {
		if ejsonPublicKey, ejsonPrivateKey, err = ejson.GenerateKeypair(); err != nil {
			log.Printf("error generating an ejson key pair: %v\n", err)
			return "", "", err
		} else if err = rewriteEjsonKeys(defaultEjsonKeydir, ejsonPublicKey, ejsonPrivateKey); err != nil {
			log.Printf("error rewriting ejson keys: %v\n", err)
			return "", "", err
		}
	} else if cr.Spec.RotateKeys == "no" {
		if ejsonPublicKey, ejsonPrivateKey, err = restoreEjsonKeysFromCluster(cr, defaultEjsonKeydir, kubeConfigPath); err != nil {
			log.Printf("error loading an ejson key pair from local storage: %v\n", err)
			return "", "", err
		}
	} else if cr.Spec.RotateKeys != "None" {
		log.Panic("rotateKeys is not one of yes/no/None")
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
		log.Printf("error restoring keys data from the cluster, error: %v\n", err)
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
