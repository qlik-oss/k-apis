package qust

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"

	"github.com/qlik-oss/k-apis/pkg/config"
	"gopkg.in/yaml.v2"
	"sigs.k8s.io/kustomize/api/types"
)

const serviceSecretKustomizationFileYaml = `apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
  - selectivepatch.yaml
transformers:
  - ../gomplate.yaml
`

const patchedSecretsGomplateFileYaml = `apiVersion: qlik.com/v1
kind: Gomplate
metadata:
  name: patched-secrets-gomplate
  labels:
    key: gomplate
dataSource:
  ejson:
    filePath: edata.json
`

func ProcessSecrets(cr *config.CRSpec, ejsonPublicKey string) error {
	baseSecretDir := filepath.Join(cr.GetManifestsRoot(), operatorPatchBaseFolder, "secrets")
	if _, err := os.Stat(baseSecretDir); os.IsNotExist(err) {
		return fmt.Errorf("%v does not exist", baseSecretDir)
	} else if err := ioutil.WriteFile(filepath.Join(baseSecretDir, "gomplate.yaml"), []byte(patchedSecretsGomplateFileYaml), os.ModePerm); err != nil {
		return errors.Wrapf(err, "error writing out the secrets' gomplate.yaml file: %v", filepath.Join(baseSecretDir, "gomplate.yaml"))
	} else if pm, err := createSupperSecretSelectivePatch(cr.Secrets); err != nil {
		return errors.Wrap(err, "error creating the selective patches map")
	} else {
		for svc, sps := range pm {
			dir := filepath.Join(baseSecretDir, svc)
			if err := addResourceToKustomization(svc, filepath.Join(baseSecretDir, "kustomization.yaml")); err != nil {
				return errors.Wrapf(err, "error adding resource: %v to kustomization file: %v", svc, filepath.Join(baseSecretDir, "kustomization.yaml"))
			} else if err := os.MkdirAll(dir, os.ModePerm); err != nil {
				return errors.Wrapf(err, "error creating directory: %v", dir)
			} else if err := ioutil.WriteFile(filepath.Join(dir, "kustomization.yaml"), []byte(serviceSecretKustomizationFileYaml), os.ModePerm); err != nil {
				return errors.Wrapf(err, "error writing out service secret kustomization.yaml file: %v", filepath.Join(dir, "kustomization.yaml"))
			} else if err := writeSelectivePatchFile(dir, sps); err != nil {
				return errors.Wrap(err, "error writing out secret selective patch")
			} else if err := writeEjsonFile(dir, cr.Secrets[svc], ejsonPublicKey); err != nil {
				return errors.Wrap(err, "error writing out secret selective patch")
			}
		}
	}
	return nil
}

func writeEjsonFile(dir string, secrets config.NameValues, ejsonPublicKey string) error {
	ejsonDataMap := make(map[string]string)
	ejsonDataMap["_public_key"] = ejsonPublicKey
	for _, secret := range secrets {
		ejsonDataMap[secret.Name] = secret.GetSecretValue()
	}
	return writeToEjsonFile(ejsonDataMap, filepath.Join(dir, "edata.json"))
}

func writeSelectivePatchFile(dir string, sps *config.SelectivePatch) error {
	if selectivePatchData, err := yaml.Marshal(sps); err != nil {
		return err
	} else {
		return ioutil.WriteFile(filepath.Join(dir, "selectivepatch.yaml"), selectivePatchData, os.ModePerm)
	}
}

// create a selectivepatch map for each service for a secretKey
func createSupperSecretSelectivePatch(sec map[string]config.NameValues) (map[string]*config.SelectivePatch, error) {
	spMap := make(map[string]*config.SelectivePatch)
	for svc, data := range sec {
		spMap[svc] = getSuperSecretSPTemplate(svc)
		for _, conf := range data {
			sp := getSuperSecretSPTemplate(svc)
			sp.Patches = []types.Patch{getSecretPatchBody(svc, conf)}
			if _, err := mergeSelectivePatches(spMap[svc], sp); err != nil {
				return nil, err
			}
		}
	}
	return spMap, nil
}

// create a patch section to be added to the selective patch
func getSecretPatchBody(svc string, nv config.NameValue) types.Patch {
	ph := getSuperSecretTemplate(svc)
	weird := "__SOMETHING_WEIRD__"
	ph.StringData = map[string]string{
		nv.Name: weird,
	}
	phb, _ := yaml.Marshal(ph)
	actual := fmt.Sprintf(`'(( (ds "data").%s | regexp.Replace "[\r\n]+" "\\n" | strings.Squote | strings.TrimPrefix "'" | strings.TrimSuffix "'" ))'`, nv.Name)
	p1 := types.Patch{
		Patch:  strings.Replace(string(phb), weird, actual, -1),
		Target: getSelector("SuperSecret", svc),
	}
	return p1
}

// a SelectivePatch object with service name in it
func getSuperSecretSPTemplate(svc string) *config.SelectivePatch {
	return getSelectivePatchTemplate(svc + "-generated-operator-secrets")
}

func getSuperSecretTemplate(svc string) *config.SupperSecret {
	return &config.SupperSecret{
		ApiVersion: "qlik.com/v1",
		Kind:       "SuperSecret",
		Metadata: map[string]string{
			"name": svc + "-secrets",
		},
	}
}
