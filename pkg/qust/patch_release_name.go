package qust

import (
	"io/ioutil"
	"log"
	"path/filepath"
	"strings"

	"github.com/qlik-oss/k-apis/pkg/config"
)

const (
	releaseTemplateFileName = "release-name-template.yaml"
)

// It will create patch for releaseName
func ProcessReleaseName(cr *config.KApiCr) error {
	if cr.GetObjectMeta().GetName() == "" {
		// no release name defined (default qliksense will be used)
		return nil
	}
	transFolder := filepath.Join(cr.Spec.GetManifestsRoot(), operatorPatchBaseFolder, "transformers")
	releaseTemplateFile := filepath.Join(transFolder, releaseTemplateFileName)
	releaseFileName := filepath.Join(transFolder, cr.GetObjectMeta().GetName()+".yaml")
	content, err := ioutil.ReadFile(releaseTemplateFile)
	if err != nil {
		log.Println("cannot read "+releaseTemplateFile, err)
		return err
	}
	result := strings.Replace(string(content), "release-template", cr.GetObjectMeta().GetName(), 1)
	result = strings.Replace(string(content), "release: qliksense", "release: "+cr.GetObjectMeta().GetName(), 1)
	if err = ioutil.WriteFile(releaseFileName, []byte(result), 0644); err != nil {
		log.Println("cannot write file " + releaseFileName)
		return err
	}
	if err = addResourceToKustomization(cr.GetObjectMeta().GetName()+".yaml", filepath.Join(transFolder, "kustomization.yaml")); err != nil {
		log.Println("Cannot process configs", err)
		return err
	}

	return nil
}
