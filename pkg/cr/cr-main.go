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

func GeneratePatches(cr *config.CRSpec, kubeConfigPath string) {
	if cr.Git.Repository == "" {
		createPatches(cr, kubeConfigPath)
	} else {
		var r *git.Repository
		var auth transport.AuthMethod
		if cr.Git.UserName != "" && cr.Git.Password != "" {
			auth = &http.BasicAuth{
				Username: cr.Git.UserName,
				Password: cr.Git.Password,
			}
		}
		if cr.Git.AccessToken != "" {
			username := cr.Git.UserName
			if username == "" {
				username = "installer"
			}
			auth = &http.BasicAuth{
				Username: username,
				Password: cr.Git.AccessToken,
			}
		}

		// Clone or open
		if _, err := os.Stat(cr.GetManifestsRoot()); os.IsNotExist(err) {
			r, err = crGit.CloneRepository(cr.GetManifestsRoot(), cr.Git.Repository, auth)
			if err != nil {
				log.Printf("error cloning repository %s: %v\n", cr.Git.Repository, err)
				return
			}
		} else {
			r, err = crGit.OpenRepository(cr.GetManifestsRoot())
			if err != nil {
				log.Printf("error opening repository %s: %v\n", cr.Git.Repository, err)
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
		err = crGit.AddCommit(r, cr.Git.UserName)
		if err != nil {
			log.Printf("error adding commit: %v\n", err)
			return
		}
		//push patches
		err = crGit.Push(r, auth)
		if err != nil {
			log.Printf("error pushing to %s: %v\n", cr.Git.Repository, err)
			return
		}
		//create pr
		// err = crGit.CreatePR(cr.GetManifestsRoot(), cr.Git.AccessToken, cr.Git.UserName, toBranch)
		// if err != nil {
		// 	log.Printf("error creating pr against %s: %v\n", cr.Git.Repository, err)

		// }
	}
}

func createPatches(cr *config.CRSpec, kubeConfigPath string) {
	//process cr.releaseName
	qust.ProcessReleaseName(cr)
	// process cr.storageClassName
	if cr.StorageClassName != "" {
		qust.ProcessStorageClassName(cr)
		// added to the configs so that down the road it is being processed
		cr.AddToConfigs("qliksense", "storageClassName", cr.StorageClassName)
	}
	// process cr.Namespace
	qust.ProcessNamespace(cr)

	// Process cr.configs
	qust.ProcessConfigs(cr)
	// Process cr.secrets
	qust.ProcessSecrets(cr)

	switch cr.RotateKeys {
	case "yes":
		generateKeys(cr, defaultEjsonKeydir)
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

func backupKeys(cr *config.CRSpec, defaultKeyDir string, kubeConfigPath string) {
	log.Println("backing up keys into the cluster")
	if err := state.Backup(kubeConfigPath, backupConfigMapName, cr.NameSpace, cr.ReleaseName, []state.BackupDir{
		{ConfigmapKey: "operator-keys", Directory: filepath.Join(cr.GetManifestsRoot(), ".operator/keys")},
		{ConfigmapKey: "ejson-keys", Directory: getEjsonKeyDir(defaultKeyDir)},
	}); err != nil {
		log.Printf("error backing up keys data to the cluster, error: %v\n", err)
	}
}

func restoreKeys(cr *config.CRSpec, defaultKeyDir string, kubeConfigPath string) {
	log.Println("restoring keys from the cluster")
	if err := state.Restore(kubeConfigPath, backupConfigMapName, cr.NameSpace, []state.BackupDir{
		{ConfigmapKey: "operator-keys", Directory: filepath.Join(cr.GetManifestsRoot(), ".operator/keys")},
		{ConfigmapKey: "ejson-keys", Directory: getEjsonKeyDir(defaultKeyDir)},
	}); err != nil {
		log.Printf("error restoring keys data from the cluster, error: %v\n", err)
	}
}
