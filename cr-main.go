package cr

import (
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"

	"github.com/Shopify/ejson"
	"github.com/qlik-oss/k-apis/config"
	"github.com/qlik-oss/k-apis/qust"
	"github.com/qlik-oss/k-apis/state"
)

const (
	defaultEjsonKeydir  = "/opt/ejson/keys"
	kubeConfigPath      = "/root/.kube/config"
	backupConfigMapName = "qliksense-operator-state-backup"
)

func GeneratePatches(cr *config.CRConfig) {
	if cr.Git.Repository == "" {
		createPatches(cr)
	} else {
		// TODO: add git pull functionality
		log.Println("Download from git repo and then call createPatches")
	}
}

func createPatches(cr *config.CRConfig) {
	// process cr.storageClassName
	if cr.StorageClassName != "" {
		qust.ProcessStorageClassName(cr)
		// added to the configs so that down the road it is being processed
		cr.AddToConfigs("qliksense", "storageClassName", cr.StorageClassName)
	}
	// process cr.Namespace
	qust.ProcessNamespace(cr)

	// Process cr.configs
	qust.ProcessCrConfigs(cr)
	// Process cr.secrets
	qust.ProcessCrSecrets(cr)

	switch cr.RotateKeys {
	case "yes":
		generateKeys(cr, defaultEjsonKeydir)
		backupKeys(cr, defaultEjsonKeydir)
	case "None":
		log.Println("no keys operations, use default EJSON_KEY")
	default:
		restoreKeys(cr, defaultEjsonKeydir)
	}
}

func generateKeys(cr *config.CRConfig, defaultKeyDir string) {
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

func backupKeys(cr *config.CRConfig, defaultKeyDir string) {
	log.Println("backing up keys into the cluster")
	if err := state.Backup(kubeConfigPath, backupConfigMapName, cr.NameSpace, []state.BackupDir{
		{ConfigmapKey: "operator-keys", Directory: filepath.Join(cr.ManifestsRoot, ".operator/keys")},
		{ConfigmapKey: "ejson-keys", Directory: getEjsonKeyDir(defaultKeyDir)},
	}); err != nil {
		log.Printf("error backing up keys data to the cluster, error: %v\n", err)
	}
}

func restoreKeys(cr *config.CRConfig, defaultKeyDir string) {
	log.Println("restoring keys from the cluster")
	if err := state.Restore(kubeConfigPath, backupConfigMapName, cr.NameSpace, []state.BackupDir{
		{ConfigmapKey: "operator-keys", Directory: filepath.Join(cr.ManifestsRoot, ".operator/keys")},
		{ConfigmapKey: "ejson-keys", Directory: getEjsonKeyDir(defaultKeyDir)},
	}); err != nil {
		log.Printf("error restoring keys data from the cluster, error: %v\n", err)
	}
}
