package qust

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/qlik-oss/k-apis/pkg/config"
)

// It will enable storageClassName transformer
func ProcessStorageClassName(cr *config.CRSpec) error {
	if cr.StorageClassName == "" {
		// no storage class defined
		return nil
	}
	storageClassTemplateName := filepath.Join(cr.GetManifestsRoot(), operatorPatchBaseFolder, "transformers", "storage-class-template.yaml")
	if _, err := os.Stat(storageClassTemplateName); os.IsNotExist(err) {
		log.Panic(storageClassTemplateName + " does not exist ")
		return err
	}
	return enableStorageClassNameTransformer(storageClassTemplateName, cr.StorageClassName, cr.GetManifestsRoot())
}

func enableStorageClassNameTransformer(storageClassFileName string, storageClassName string, manifestsRoot string) error {
	fileContents, err := ioutil.ReadFile(storageClassFileName)
	if err != nil {
		return err
	}
	replaceContents := strings.Replace(string(fileContents), "value: false", "value: true", -1)
	storageClassReleaseName := filepath.Join(manifestsRoot, operatorPatchBaseFolder, "transformers", storageClassName+".yaml")
	if err = ioutil.WriteFile(storageClassReleaseName, []byte(replaceContents), 0644); err != nil {
		log.Println("cannot write file " + storageClassReleaseName)
		return err
	}
	if err := addResourceToKustomization(storageClassReleaseName, storageClassFileName); err != nil {
		log.Println("Cannot create storage class", err)
		return err
	}
	return nil
}
