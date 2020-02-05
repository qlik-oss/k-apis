package qust

import (
	"fmt"
	"github.com/qlik-oss/k-apis/pkg/config"
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

// It will create patch for releaseName
func ProcessReleaseName(cr *config.CRSpec) error {
	if cr.ReleaseName == "" {
		// no release name defined (default qliksense will be used)
		return nil
	}
	releaseFileName := filepath.Join(cr.GetManifestsRoot(), operatorPatchBaseFolder, "transformers", "release-name.yaml")
	if _, err := os.Stat(releaseFileName); os.IsNotExist(err) {
		log.Panic(releaseFileName + " does not exist ")
		return err
	}
	return changeReleaseName(cr.ReleaseName, releaseFileName)
}

func changeReleaseName(releaseName, releaseFileName string) error {
	//sed -i -e 's/release\: qliksense/release\: new-release-name/g' release-name.yaml
	s := `s/release\: qliksense/release\: %s/g`
	cmd := exec.Command("sed", "-i", "-e", fmt.Sprintf(s, releaseName), releaseFileName)
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
		return err
	}
	return nil

}
