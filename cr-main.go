package cr

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/Shopify/ejson"
	"github.com/google/go-github/github"
	"github.com/qlik-oss/k-apis/config"
	"github.com/qlik-oss/k-apis/qust"
	"github.com/qlik-oss/k-apis/state"
	"golang.org/x/oauth2"

	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"gopkg.in/src-d/go-git.v4/plumbing/transport/http"
)

const (
	defaultEjsonKeydir  = "/opt/ejson/keys"
	kubeConfigPath      = "/root/.kube/config"
	backupConfigMapName = "qliksense-operator-state-backup"
)

func GeneratePatches(cr *config.CRSpec) {
	if cr.Git.Repository == "" {
		createPatches(cr)
	} else {
		log.Println("Download from git repo and then call createPatches")

		//Open Git Repositorty
		r, err := git.PlainOpen(cr.Git.Repository)
		if err != nil {
			log.Printf("error opening repository: %v\n", err)
			return
		}
		w, err := r.Worktree()
		if err != nil {
			log.Printf("error getting working tree: %v\n", err)
			return
		}

		err = w.Checkout(&git.CheckoutOptions{
			Branch: "refs/heads/master",
			Force:  true,
		})
		if err != nil {
			log.Printf("error setting reference to new branch: %v\n", err)
		}

		err = w.Pull(&git.PullOptions{RemoteName: "origin"})
		if err != git.NoErrAlreadyUpToDate && err != nil {
			log.Printf("error pulling working tree: %v\n", err)
		}
		headRef, err := r.Head()
		if err != nil {
			log.Printf("error setting head: %v\n", err)
		}

		reference := plumbing.NewBranchReferenceName(
			fmt.Sprintf("refs/heads/pr-branch-%s", tokenGenerator()),
		)
		fmt.Println(reference)
		ref := plumbing.NewHashReference(reference, headRef.Hash())

		err = r.Storer.SetReference(ref)
		if err != nil {
			log.Printf("error setting reference to new branch: %v\n", err)
		}
		err = w.Checkout(&git.CheckoutOptions{
			Branch: ref.Name(),
			Force:  true,
		})
		if err != nil {
			log.Printf("error setting reference to new branch: %v\n", err)
		}
		makePrWithPatches(cr, r, w, ref.Name())
	}
}

func makePrWithPatches(cr *config.CRConfig, r *git.Repository, w *git.Worktree, branch plumbing.ReferenceName) {

	//createPatches(cr)

	filename := filepath.Join(cr.Git.Repository, "example-git-file")
	err := ioutil.WriteFile(filename, []byte("hello world5!"), 0644)

	_, err = w.Add(".")

	_, err = w.Commit("k-apis pr", &git.CommitOptions{
		Author: &object.Signature{
			Name: cr.Git.UserName,
			When: time.Now(),
		},
	})

	err = r.Push(&git.PushOptions{
		Auth: &http.BasicAuth{
			Username: cr.Git.UserName,
			Password: cr.Git.Password,
		},
	})
	if err != nil {
		log.Printf("error creating push: %v\n", err)
		return
	}

	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: cr.Git.AccessToken},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	newPR := &github.NewPullRequest{
		Title:               github.String("k-apis PR"),
		Head:                github.String(branch.String()),
		Base:                github.String("master"),
		Body:                github.String("auto generated pr"),
		MaintainerCanModify: github.Bool(true),
	}

	_, _, err = client.PullRequests.Create(context.Background(), cr.Git.UserName, "golang-server", newPR)
	if err != nil {
		log.Printf("error creating pr: %v\n", err)
		return
	}
}

func createPatches(cr *config.CRSpec) {
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
		backupKeys(cr, defaultEjsonKeydir)
	case "None":
		log.Println("no keys operations, use default EJSON_KEY")
	default:
		restoreKeys(cr, defaultEjsonKeydir)
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

func backupKeys(cr *config.CRSpec, defaultKeyDir string) {
	log.Println("backing up keys into the cluster")
	if err := state.Backup(kubeConfigPath, backupConfigMapName, cr.NameSpace, []state.BackupDir{
		{ConfigmapKey: "operator-keys", Directory: filepath.Join(cr.ManifestsRoot, ".operator/keys")},
		{ConfigmapKey: "ejson-keys", Directory: getEjsonKeyDir(defaultKeyDir)},
	}); err != nil {
		log.Printf("error backing up keys data to the cluster, error: %v\n", err)
	}
}

func restoreKeys(cr *config.CRSpec, defaultKeyDir string) {
	log.Println("restoring keys from the cluster")
	if err := state.Restore(kubeConfigPath, backupConfigMapName, cr.NameSpace, []state.BackupDir{
		{ConfigmapKey: "operator-keys", Directory: filepath.Join(cr.ManifestsRoot, ".operator/keys")},
		{ConfigmapKey: "ejson-keys", Directory: getEjsonKeyDir(defaultKeyDir)},
	}); err != nil {
		log.Printf("error restoring keys data from the cluster, error: %v\n", err)
	}
}
