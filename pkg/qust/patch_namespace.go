package qust

import (
	"io/ioutil"
	"log"
	"path/filepath"
	"strings"

	"github.com/qlik-oss/k-apis/pkg/config"
)

// It will patch the built-in NamespaceTransformer
func ProcessNamespace(cr *config.KApiCr) error {
	if cr.GetObjectMeta().GetNamespace() == "" {
		// no namespace provided so default should work
		return nil
	}
	namespacePatchFileName := "namespace-" + cr.GetObjectMeta().GetNamespace() + ".yaml"

	fileFullPath := filepath.Join(cr.Spec.GetManifestsRoot(), operatorPatchBaseFolder, "transformers", namespacePatchFileName)
	fileContents := strings.Replace(namespacePatchTemplate(), "NAMESPACE_NAME", cr.GetObjectMeta().GetNamespace(), 1)

	err := ioutil.WriteFile(fileFullPath, []byte(fileContents), FILE_PERMISION)

	if err != nil {
		log.Panic("Cannnot create patch for namespace ", err)
		return err
	}
	// add that file to kustomization.yaml
	fileFullPath = filepath.Join(cr.Spec.GetManifestsRoot(), operatorPatchBaseFolder, "transformers", "kustomization.yaml")
	err = addResourceToKustomization(namespacePatchFileName, fileFullPath)
	if err != nil {
		log.Panic("Cannot add resource to "+fileFullPath, err)
		return err
	}

	return nil
}

func namespacePatchTemplate() string {
	return `
apiVersion: qlik.com/v1
kind: SelectivePatch
metadata:
  name: operator-patch-for-namespace
enabled: true
patches:
- target:
    kind: NamespaceTransformer
  patch: |-
    - op: replace
      path: /metadata/namespace
      value: NAMESPACE_NAME
`
}
