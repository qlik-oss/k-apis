package cr

import (
	"log"

	"github.com/qlik-oss/k-apis/pkg/config"
	"github.com/qlik-oss/k-apis/pkg/qust"
)

const (
	defaultEjsonKeydir      = "/opt/ejson/keys"
	defaultBackupObjectName = "operator-state-backup"
)

func GeneratePatches(cr *config.KApiCr, keysAction config.KeysAction, kubeConfigPath string) {
	if err := createPatches(cr, keysAction, kubeConfigPath); err != nil {
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
func createPatches(cr *config.KApiCr, keysAction config.KeysAction, kubeConfigPath string) error {
	if keysAction != config.KeysActionForceRotate && keysAction != config.KeysActionDoNothing {
		keysAction = config.KeysActionRestoreOrRotate
	}

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

	// regenerate the ejson key pair, or restore it from the cluster, or read it from the environment
	ejsonPublicKey, _, err := processEjsonKeys(cr, keysAction, kubeConfigPath, defaultEjsonKeydir)
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
	if err := finalizeKeys(cr, keysAction, kubeConfigPath, ejsonPublicKey); err != nil {
		return err
	}

	return nil
}
