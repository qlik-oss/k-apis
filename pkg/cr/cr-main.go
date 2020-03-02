package cr

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"

	"github.com/Shopify/ejson"
	"github.com/qlik-oss/k-apis/pkg/config"
	crGit "github.com/qlik-oss/k-apis/pkg/git"
	"github.com/qlik-oss/k-apis/pkg/qust"
	"github.com/qlik-oss/k-apis/pkg/state"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing/transport"
	"gopkg.in/src-d/go-git.v4/plumbing/transport/http"
)

const (
	defaultEjsonKeydir  = "/opt/ejson/keys"
	backupConfigMapName = "qliksense-operator-state-backup"
)

func GeneratePatches(cr *config.KApiCr, kubeConfigPath string) {
	if cr.Spec.Git.Repository == "" {
		createPatches(cr, kubeConfigPath)
	} else {
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

		createPatches(cr, kubeConfigPath)
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
}

func createPatches(cr *config.KApiCr, kubeConfigPath string) {
	//process cr.releaseName
	qust.ProcessReleaseName(cr)
	// process cr.storageClassName
	if cr.Spec.StorageClassName != "" {
		qust.ProcessStorageClassName(cr.Spec)
		// added to the configs so that down the road it is being processed
		cr.Spec.AddToConfigs("qliksense", "storageClassName", cr.Spec.StorageClassName)
	}
	// process cr.Namespace
	qust.ProcessNamespace(cr)

	// Process cr.configs
	qust.ProcessConfigs(cr.Spec)
	// Process cr.secrets
	qust.ProcessSecrets(cr.Spec)

	switch cr.Spec.RotateKeys {
	case "yes":
		generateKeys(cr.Spec, defaultEjsonKeydir)
		backupKeys(cr, defaultEjsonKeydir, kubeConfigPath)
	case "None":
		log.Println("no keys operations, use default EJSON_KEY")
	default:
		restoreKeys(cr, defaultEjsonKeydir, kubeConfigPath)
	}
}

func generateKeys(cr *config.CRSpec, defaultKeyDir string) {
	log.Println("rotating all keys")
	keyDir := getEjsonKeyDir(defaultKeyDir)
	if ejsonPublicKey, ejsonPrivateKey, err := ejson.GenerateKeypair(); err != nil {
		log.Printf("error generating an ejson key pair: %v\n", err)
	} else if err := qust.GenerateKeys(cr, ejsonPublicKey); err != nil {
		log.Printf("error generating application keys: %v\n", err)
	} else if err := os.MkdirAll(keyDir, os.ModePerm); err != nil {
		log.Printf("error makeing sure private key storage directory: %v exists, error: %v\n", keyDir, err)
	} else if err := ioutil.WriteFile(path.Join(keyDir, ejsonPublicKey), []byte(ejsonPrivateKey), os.ModePerm); err != nil {
		log.Printf("error storing ejson private key: %v\n", err)
	}
}

func getEjsonKeyDir(defaultKeyDir string) string {
	ejsonKeyDir := os.Getenv("EJSON_KEYDIR")
	if ejsonKeyDir == "" {
		ejsonKeyDir = defaultKeyDir
	}
	return ejsonKeyDir
}

func backupKeys(cr *config.KApiCr, defaultKeyDir string, kubeConfigPath string) {
	log.Println("backing up keys into the cluster")
	if err := state.Backup(kubeConfigPath, backupConfigMapName, cr.GetObjectMeta().GetNamespace(), cr.GetObjectMeta().GetName(), []state.BackupDir{
		{ConfigmapKey: "operator-keys", Directory: filepath.Join(cr.Spec.GetManifestsRoot(), ".operator/keys")},
		{ConfigmapKey: "ejson-keys", Directory: getEjsonKeyDir(defaultKeyDir)},
	}); err != nil {
		log.Printf("error backing up keys data to the cluster, error: %v\n", err)
	}
}

func restoreKeys(cr *config.KApiCr, defaultKeyDir string, kubeConfigPath string) {
	log.Println("restoring keys from the cluster")
	if err := state.Restore(kubeConfigPath, backupConfigMapName, cr.GetObjectMeta().GetNamespace(), []state.BackupDir{
		{ConfigmapKey: "operator-keys", Directory: filepath.Join(cr.Spec.GetManifestsRoot(), ".operator/keys")},
		{ConfigmapKey: "ejson-keys", Directory: getEjsonKeyDir(defaultKeyDir)},
	}); err != nil {
		log.Printf("error restoring keys data from the cluster, error: %v\n", err)
	}
}
