package qust

import (
	"errors"
	"io"
	"io/ioutil"

	"github.com/qlik-oss/k-apis/pkg/config"
	"gopkg.in/yaml.v2"

	"sigs.k8s.io/kustomize/api/resid"
	"sigs.k8s.io/kustomize/api/types"
)

const (
	// a folder named .operator exist in manifestsRoot
	operatorPatchBaseFolder = ".operator"
	FILE_PERMISION          = 0644
)

// merge two selective patch by appending sp2.patches into sp1.patch
// the final result is sp1.patchs updated
func mergeSelectivePatches(sp1, sp2 *config.SelectivePatch) (*config.SelectivePatch, error) {
	if sp1 == nil && sp2 == nil {
		return nil, errors.New("Two objects are nil")
	}
	if sp1 == nil {
		return sp2, nil
	}
	if sp2 == nil {
		return sp1, nil
	}
	if sp1.Kind != sp2.Kind || sp1.Metadata.Name != sp2.Metadata.Name {
		err := errors.New("Cannot merge selective patches [ " + sp1.Metadata.Name + " != " + sp2.Metadata.Name)
		return nil, err
	}
	sp1.Patches = append(sp1.Patches, sp2.Patches...)
	return sp1, nil
}

func YamlToWriter(w io.Writer, yml interface{}) error {
	d, err := yaml.Marshal(yml)
	w.Write(d)
	return err
}

// add a resource file in kustomization if not that exist
func addResourceToKustomization(rsFileName string, kustFile string) error {
	fn := func(kust *types.Kustomization) {
		// if the resource exist no need to add again
		if !isResourcesInKust(rsFileName, kust) {
			kust.Resources = append(kust.Resources, rsFileName)
		}
	}
	return kustFileHelper(kustFile, fn)
}

// it is a helper to add any file as a resource,transfomer, generator, etc
// fn will define what type of file it would be
func kustFileHelper(kustFile string, fn func(*types.Kustomization)) error {
	kust := &types.Kustomization{}
	content, err := ioutil.ReadFile(kustFile)
	if err != nil {
		return err
	}
	yaml.Unmarshal(content, kust)

	fn(kust)

	kust.FixKustomizationPostUnmarshalling()
	// there is a bug if not put nil https://github.com/kubernetes-sigs/kustomize/pull/1004/files

	d, err := yaml.Marshal(kust)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(kustFile, d, FILE_PERMISION)
	return err
}

func isResourcesInKust(rsFileName string, kust *types.Kustomization) bool {
	for _, k := range kust.Resources {
		if k == rsFileName {
			return true
		}
	}
	return false
}

func getSelector(kind, svc string) *types.Selector {
	if svc == "" {
		return &types.Selector{
			Gvk: resid.Gvk{
				Kind: kind,
			}}
	}
	return &types.Selector{
		Gvk: resid.Gvk{
			Kind: kind,
		},
		LabelSelector: "app=" + svc,
	}
}

// a SelectivePatch object with service name in it
func getSelectivePatchTemplate(name string) *config.SelectivePatch {
	su := &config.SelectivePatch{
		ApiVersion: "qlik.com/v1",
		Kind:       "SelectivePatch",
		Metadata: &config.CustomMetadata{
			Name: name,
		},
		Enabled: false,
	}
	return su
}

func getResourcesList(kustFile string) ([]string, error) {
	kust := &types.Kustomization{}
	by, err := ioutil.ReadFile(kustFile)
	if err != nil {
		return nil, err
	}
	if err := yaml.Unmarshal(by, kust); err != nil {
		return nil, err
	}
	return kust.Resources, nil
}

func contains(arr []string, str string) bool {
	for _, a := range arr {
		if a == str {
			return true
		}
	}
	return false
}

func removeResourceFromKust(rsName, kustFile string) error {
	fn := func(kust *types.Kustomization) {
		newRes := make([]string, 0)
		// if the resource exist remove it
		for _, r := range kust.Resources {
			if r != rsName && r != "" && rsName != "" {
				newRes = append(newRes, r)
			}
		}
		kust.Resources = newRes
	}
	return kustFileHelper(kustFile, fn)
}
