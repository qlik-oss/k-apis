package qust

import (
	"github.com/qlik-oss/k-apis/pkg/config"
	"io/ioutil"
	"log"
	"path/filepath"
	"strings"
)

const (
	releaseTemplateFileName = "release-name-template.yaml"
)

// It will create patch for releaseName
func ProcessReleaseName(cr *config.CRSpec) error {
	if cr.ReleaseName == "" {
		// no release name defined (default qliksense will be used)
		return nil
	}
	transFolder := filepath.Join(cr.GetManifestsRoot(), operatorPatchBaseFolder, "transformers")
	releaseTemplateFile := filepath.Join(transFolder, releaseTemplateFileName)
	releaseFileName := filepath.Join(transFolder, cr.ReleaseName+".yaml")
	content, err := ioutil.ReadFile(releaseTemplateFile)
	if err != nil {
		log.Println("cannot read "+releaseTemplateFile, err)
		return err
	}
	result := strings.Replace(string(content), "release-template", cr.ReleaseName, 1)
	result = strings.Replace(string(content), "release: qliksense", "release: "+cr.ReleaseName, 1)
	if err = ioutil.WriteFile(releaseFileName, []byte(result), 0644); err != nil {
		log.Println("cannot write file " + releaseFileName)
		return err
	}
	if err = addResourceToKustomization(cr.ReleaseName+".yaml", filepath.Join(transFolder, "kustomization.yaml")); err != nil {
		log.Println("Cannot process configs", err)
		return err
	}

	return nil
}
