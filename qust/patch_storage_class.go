package qust

import (
	"github.com/qlik-oss/k-apis/config"
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

// It will enable storageClassName transformer
func ProcessStorageClassName(cr *config.CRSpec) error {
	if cr.StorageClassName == "" {
		// no storage class defined
		return nil
	}
	storageClassFileName := filepath.Join(cr.ManifestsRoot, operatorPatchBaseFolder, "transformers", "storage-class.yaml")
	if _, err := os.Stat(storageClassFileName); os.IsNotExist(err) {
		log.Panic(storageClassFileName + " does not exist ")
		return err
	}
	return enableStorageClassNameTransformer(storageClassFileName)
}

func enableStorageClassNameTransformer(storageClassFileName string) error {
	//sed -i -e 's/value\: false/value\: true/g' storage-class.yaml
	s := `s/value\: false/value\: true/g`
	cmd := exec.Command("sed", "-i", "-e", s, storageClassFileName)
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
		return err
	}
	return nil
}
