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
	storageClassFileName := filepath.Join(cr.GetManifestsRoot(), operatorPatchBaseFolder, "transformers", "storage-class.yaml")
	if _, err := os.Stat(storageClassFileName); os.IsNotExist(err) {
		log.Panic(storageClassFileName + " does not exist ")
		return err
	}
	return enableStorageClassNameTransformer(storageClassFileName)
}

func enableStorageClassNameTransformer(storageClassFileName string) error {
	//sed -i -e 's/value\: false/value\: true/g' storage-class.yaml
	fileContents, err := ioutil.ReadFile(storageClassFileName)
	if err != nil {
		log.Fatal(err)
		return err
	}
	replaceContents := strings.Replace(string(fileContents), "value: false", "value: true", -1)
	return ioutil.WriteFile(storageClassFileName, []byte(replaceContents), 0644)
}
